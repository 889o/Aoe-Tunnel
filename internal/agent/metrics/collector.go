// Package metrics 采集 Agent 所在主机的资源快照，仅依赖 /proc 与 statfs，不引入额外依赖。
package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"aoe-tunnel/pkg/signal"
)

type cpuSample struct {
	total uint64
	idle  uint64
}

type netSample struct {
	in  uint64
	out uint64
	at  time.Time
}

type Collector struct {
	log *zap.Logger

	mu   sync.Mutex
	cpu  cpuSample
	net  netSample
	seen bool
}

func New(log *zap.Logger) *Collector {
	return &Collector{log: log}
}

// Collect 返回自上次调用以来的 CPU/网卡利用率与当前资源占用；首次调用缺少基准按 0 处理。
func (c *Collector) Collect() (signal.SystemMetrics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	m := signal.SystemMetrics{TS: time.Now().Unix()}

	cpu, err := readCPU()
	if err != nil {
		return m, fmt.Errorf("metrics: cpu: %w", err)
	}
	if c.seen && cpu.total > c.cpu.total {
		dTotal := float64(cpu.total - c.cpu.total)
		dIdle := float64(cpu.idle - c.cpu.idle)
		if dTotal > 0 {
			m.CPUPercent = clampPercent((dTotal - dIdle) / dTotal * 100)
		}
	}
	c.cpu = cpu

	if err := readMemory(&m); err != nil {
		return m, fmt.Errorf("metrics: memory: %w", err)
	}
	readLoad(&m)
	readUptime(&m)
	readDisk(&m)
	if err := c.readNet(&m); err != nil {
		c.log.Debug("metrics: net stats unavailable", zap.Error(err))
	}

	c.seen = true
	return m, nil
}

func readCPU() (cpuSample, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuSample{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)[1:]
		var total, idle uint64
		for i, raw := range fields {
			v, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				continue
			}
			total += v
			// idle 与 iowait 都算作空闲时间，口径与 top 一致。
			if i == 3 || i == 4 {
				idle += v
			}
		}
		return cpuSample{total: total, idle: idle}, nil
	}
	if err := sc.Err(); err != nil {
		return cpuSample{}, err
	}
	return cpuSample{}, fmt.Errorf("cpu line not found")
}

func readMemory(m *signal.SystemMetrics) error {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	var totalKB, availKB uint64
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		v, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch strings.TrimSuffix(fields[0], ":") {
		case "MemTotal":
			totalKB = v
		case "MemAvailable":
			availKB = v
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if totalKB == 0 {
		return fmt.Errorf("meminfo total missing")
	}
	m.MemTotal = totalKB * 1024
	if availKB < totalKB {
		m.MemUsed = (totalKB - availKB) * 1024
	}
	m.MemPercent = clampPercent(float64(m.MemUsed) / float64(m.MemTotal) * 100)
	return nil
}

func readLoad(m *signal.SystemMetrics) {
	raw, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 3 {
		return
	}
	m.Load1, _ = strconv.ParseFloat(fields[0], 64)
	m.Load5, _ = strconv.ParseFloat(fields[1], 64)
	m.Load15, _ = strconv.ParseFloat(fields[2], 64)
}

func readUptime(m *signal.SystemMetrics) {
	raw, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return
	}
	fields := strings.Fields(string(raw))
	if len(fields) == 0 {
		return
	}
	up, _ := strconv.ParseFloat(fields[0], 64)
	m.UptimeSec = int64(up)
}

func readDisk(m *signal.SystemMetrics) {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return
	}
	bsize := uint64(st.Bsize)
	m.DiskTotal = st.Blocks * bsize
	free := st.Bavail * bsize
	if free < m.DiskTotal {
		m.DiskUsed = m.DiskTotal - free
	}
	if m.DiskTotal > 0 {
		m.DiskPercent = clampPercent(float64(m.DiskUsed) / float64(m.DiskTotal) * 100)
	}
}

func (c *Collector) readNet(m *signal.SystemMetrics) error {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	// 前两行是表头。
	sc.Scan()
	sc.Scan()

	var in, out uint64
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		iface := strings.TrimSpace(line[:idx])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(line[idx+1:])
		// 每行 16 列：接收 8（字节在第 1 列），发送 8（字节在第 9 列）。
		if len(fields) < 16 {
			continue
		}
		rx, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		tx, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			continue
		}
		in += rx
		out += tx
	}
	if err := sc.Err(); err != nil {
		return err
	}

	now := time.Now()
	if !c.net.at.IsZero() {
		sec := now.Sub(c.net.at).Seconds()
		if sec > 0 {
			if in >= c.net.in {
				m.NetInBps = int64(float64(in-c.net.in) / sec)
			}
			if out >= c.net.out {
				m.NetOutBps = int64(float64(out-c.net.out) / sec)
			}
		}
	}
	c.net = netSample{in: in, out: out, at: now}
	return nil
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
