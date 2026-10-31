package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"aoe-tunnel/internal/agent/config"
	"aoe-tunnel/internal/agent/forward"
	"aoe-tunnel/internal/agent/metrics"
	"aoe-tunnel/internal/agent/panelapi"
	"aoe-tunnel/internal/agent/rule"
	"aoe-tunnel/internal/agent/stat"
	"aoe-tunnel/internal/agent/tunnel"
	"aoe-tunnel/pkg/logger"
	"aoe-tunnel/pkg/mqttcli"
	aoesignal "aoe-tunnel/pkg/signal"
	"aoe-tunnel/pkg/version"
)

func main() {
	// 一键安装脚本以参数形式下发；参数优先于环境变量。
	nodeCode := flag.String("node-code", "", "节点编码，等价于 AGENT_NODE_CODE")
	nodeToken := flag.String("node-token", "", "节点令牌，等价于 AGENT_NODE_TOKEN")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent: load config: %v\n", err)
		os.Exit(1)
	}
	if *nodeCode != "" {
		cfg.NodeCode = *nodeCode
	}
	if *nodeToken != "" {
		cfg.NodeToken = *nodeToken
	}
	if cfg.NodeCode == "" {
		fmt.Fprintf(os.Stderr, "agent: node code required (--node-code or AGENT_NODE_CODE)\n")
		os.Exit(1)
	}

	log, err := logger.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent: init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	// 实例 ID 持久化在数据目录：同一台机器重启保持同一身份，换机器部署则生成新实例。
	instanceID := loadInstanceID(cfg.DataDir, log)

	mgr, err := rule.NewManager(cfg.DataDir, log)
	if err != nil {
		log.Fatal("agent: open rule store", zap.Error(err))
	}
	defer func() {
		if err := mgr.Close(); err != nil {
			log.Warn("agent: close rule store", zap.Error(err))
		}
	}()
	log.Info("rule store loaded", zap.Int("rules", mgr.Count()))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	certs, err := tunnel.LoadCerts(cfg.CertFile, cfg.KeyFile, cfg.ClientCertFile, cfg.ClientKeyFile, cfg.CAFile)
	if err != nil {
		log.Fatal("agent: load certs", zap.Error(err))
	}
	pool, err := tunnel.NewClientPool(tunnel.DialConfig{
		Certs:            certs,
		SNI:              cfg.SNI,
		HandshakeTimeout: 3 * time.Second,
		Log:              log,
	})
	if err != nil {
		log.Fatal("agent: init quic client", zap.Error(err))
	}
	fallback, err := tunnel.NewFallbackPool(tunnel.FallbackConfig{
		Certs:            certs,
		SNI:              cfg.SNI,
		HandshakeTimeout: 3 * time.Second,
		Log:              log,
	})
	if err != nil {
		log.Fatal("agent: init fallback client", zap.Error(err))
	}

	client := panelapi.New(panelapi.Config{
		BaseURL:   cfg.PanelAPIURL,
		Username:  cfg.PanelUser,
		Password:  cfg.PanelPass,
		NodeToken: cfg.NodeToken,
		Timeout:   5 * time.Second,
	}, log)
	hops := tunnel.NewManager(pool, fallback, cfg.ForceFallback, 3*time.Second, log)
	exitSrv, err := forward.NewExitServer(cfg.QUICAddr, cfg.TLSAddr, cfg.NodeCode, certs, mgr, log)
	if err != nil {
		log.Fatal("agent: init exit server", zap.Error(err))
	}
	go exitSrv.Run(ctx)

	fwd := forward.NewForwarder(cfg.NodeCode, cfg.ListenHost, hops, log)
	mgr.SetHooks(fwd.Apply, fwd.Remove)
	// 启动时先按 bbolt 里的规则拉起入口监听，MQTT 同步只处理增量。
	for _, r := range mgr.Snapshot() {
		fwd.Apply(nil, r)
	}

	report := func(typ string, payload any) {
		env, err := aoesignal.NewEnvelope(typ, cfg.NodeCode, payload)
		if err != nil {
			log.Warn("build report failed", zap.String("type", typ), zap.Error(err))
			return
		}
		rctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Report(rctx, cfg.NodeCode, env); err != nil {
			log.Debug("report failed", zap.String("type", typ), zap.Error(err))
		}
	}

	cli, err := mqttcli.New(mqttcli.Config{
		Broker:   cfg.MQTTBroker,
		CAFile:   cfg.MQTTCACert,
		SNI:      cfg.MQTTSNI,
		User:     cfg.MQTTUser,
		Pass:     cfg.MQTTPass,
		ClientID: "agent-" + cfg.NodeCode + "-" + uuid.NewString()[:6],
		Will: &mqttcli.Message{
			Topic:   aoesignal.StateTopic(cfg.NodeCode),
			Payload: statePayload(false, instanceID),
			Retain:  true,
		},
	}, log, func(topic string, payload []byte) {
		var env aoesignal.Envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			log.Warn("cmd decode failed", zap.String("topic", topic), zap.Error(err))
			return
		}
		// 被同节点的更新部署抢占后主动退出，保证一个入口节点只留一台服务器在线。
		if env.Type == aoesignal.TypeNodeKick {
			var kick aoesignal.NodeKick
			if err := json.Unmarshal(env.Payload, &kick); err != nil {
				log.Warn("node kick decode failed", zap.Error(err))
				return
			}
			if kick.InstanceID == instanceID {
				log.Warn("node taken over by another instance, shutting down",
					zap.String("node_code", cfg.NodeCode), zap.String("instance_id", instanceID))
				stop()
			}
			return
		}
		if err := mgr.HandleCmd(env); err != nil {
			log.Warn("cmd apply failed", zap.String("type", env.Type), zap.Error(err))
		}
	}, []string{aoesignal.CmdTopic(cfg.NodeCode)}, func(c *mqttcli.Client) {
		if err := c.PublishRetained(aoesignal.StateTopic(cfg.NodeCode), statePayload(true, instanceID)); err != nil {
			log.Warn("publish online state failed", zap.Error(err))
		}
		// 控制面只保留状态遗嘱，规则同步请求改走 JWT 上报。
		report(aoesignal.TypeNodeSyncRequest, nil)
	})
	if err != nil {
		log.Fatal("agent: init mqtt", zap.Error(err))
	}

	agg := stat.NewAggregator(mgr, func() []stat.RuleStat {
		stats := fwd.Stats()
		out := make([]stat.RuleStat, 0, len(stats))
		for _, s := range stats {
			out = append(out, stat.RuleStat{
				RuleID:      s.RuleID,
				ActiveConns: s.ActiveConns,
				UpBytes:     s.UpBytes,
				DownBytes:   s.DownBytes,
			})
		}
		return out
	}, report, fwd.QuotaExhausted, cfg.ReportInterval, log)
	go agg.Run(ctx)

	collector := metrics.New(log)
	go metricsLoop(ctx, collector, report, cfg.MetricsInterval, log)

	log.Info("agent starting",
		zap.String("node_code", cfg.NodeCode),
		zap.String("version", version.Version),
		zap.String("quic_addr", cfg.QUICAddr),
		zap.String("tls_addr", cfg.TLSAddr),
		zap.Int("loaded_rules", mgr.Count()),
		zap.Bool("force_fallback", cfg.ForceFallback),
		zap.Duration("metrics_interval", cfg.MetricsInterval),
	)

	// 节点令牌只用于上线校验；业务上报统一走 JWT。
	ipResolver := panelapi.NewPublicIPResolver(cfg.PublicIPURL)
	onlineInfo := panelapi.OnlineInfo{
		Version:    version.Version,
		QUICAddr:   cfg.QUICAddr,
		TLSAddr:    cfg.TLSAddr,
		IP:         ipResolver.Get(ctx),
		InstanceID: instanceID,
	}
	onlineCtx, onlineCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := client.Online(onlineCtx, cfg.NodeCode, onlineInfo); err != nil {
		log.Warn("node online register failed", zap.Error(err))
	}
	onlineCancel()

	if err := cli.Connect(); err != nil {
		log.Warn("mqtt initial connect failed", zap.Error(err))
	}

	startedAt := time.Now()
	go heartbeat(ctx, client, ipResolver, agg, cfg.NodeCode, onlineInfo, startedAt, log)

	<-ctx.Done()
	log.Info("agent shutting down", zap.String("node_code", cfg.NodeCode))
	fwd.Close()
	exitSrv.Close()
	fallback.Close()
	pool.Close()
	if err := cli.PublishRetained(aoesignal.StateTopic(cfg.NodeCode), statePayload(false, instanceID)); err != nil {
		log.Debug("publish offline state failed", zap.Error(err))
	}
	cli.Disconnect()
}

func heartbeat(ctx context.Context, client *panelapi.Client, ipResolver *panelapi.PublicIPResolver, agg *stat.Aggregator, nodeCode string, info panelapi.OnlineInfo, startedAt time.Time, log *zap.Logger) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 重复上线用于刷新面板记录的外网 IP，地址未变时面板不会重推规则。
			info.IP = ipResolver.Get(ctx)
			rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if err := client.Online(rctx, nodeCode, info); err != nil {
				log.Debug("node online refresh failed", zap.Error(err))
			}
			cancel()

			payload := aoesignal.NodeStatus{
				Online:      true,
				Version:     version.Version,
				UptimeSec:   int64(time.Since(startedAt).Seconds()),
				TunnelMode:  "quic+fallback",
				ActiveConns: agg.Active(),
			}
			env, err := aoesignal.NewEnvelope(aoesignal.TypeNodeStatus, nodeCode, payload)
			if err != nil {
				log.Warn("build status report failed", zap.Error(err))
				continue
			}
			rctx, cancel = context.WithTimeout(ctx, 5*time.Second)
			if err := client.Report(rctx, nodeCode, env); err != nil {
				log.Debug("status report failed", zap.Error(err))
			}
			cancel()
		}
	}
}

// metricsLoop 每秒采集一次主机资源并上报，面板只在内存缓存最近 60 个点。
func metricsLoop(ctx context.Context, collector *metrics.Collector, report func(typ string, payload any), interval time.Duration, log *zap.Logger) {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m, err := collector.Collect()
			if err != nil {
				log.Debug("metrics collect failed", zap.Error(err))
				continue
			}
			report(aoesignal.TypeNodeMetrics, m)
		}
	}
}

func statePayload(online bool, instanceID string) []byte {
	raw, _ := json.Marshal(aoesignal.StatePayload{
		Online:     online,
		Version:    version.Version,
		TS:         time.Now().Unix(),
		InstanceID: instanceID,
	})
	return raw
}

// loadInstanceID 从数据目录读取持久实例 ID；首次启动生成并落盘，重启保持同一身份。
func loadInstanceID(dataDir string, log *zap.Logger) string {
	if dataDir == "" {
		return uuid.NewString()
	}
	path := filepath.Join(dataDir, "instance.id")
	if raw, err := os.ReadFile(path); err == nil {
		if id := strings.TrimSpace(string(raw)); id != "" {
			return id
		}
	}
	_ = os.MkdirAll(dataDir, 0o755)
	id := uuid.NewString()
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		log.Warn("persist instance id failed", zap.String("path", path), zap.Error(err))
	}
	return id
}
