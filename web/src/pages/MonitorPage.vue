<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { EChartsOption } from 'echarts'
import { Activity, Cpu, HardDrive, MemoryStick, Network, RefreshCw, Server, Timer } from 'lucide-vue-next'
import { getMonitorOverview, getNodeStatHistory } from '@/api'
import type { MonitorNode, SystemMetrics } from '@/types'
import PaperBadge from '@/components/paper/PaperBadge.vue'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperCard from '@/components/paper/PaperCard.vue'
import PaperEChart from '@/components/paper/PaperEChart.vue'
import { useThemeStore } from '@/stores/theme'
import { formatBytes, formatTime } from '@/lib/utils'

const theme = useThemeStore()
const nodes = ref<MonitorNode[]>([])
const history = ref<SystemMetrics[]>([])
const selected = ref('')
const loading = ref(false)
const detailLoading = ref(false)

let timer: number | undefined

const POLL_MS = 1000
const selectedNode = computed(() => nodes.value.find((n) => n.code === selected.value) ?? null)

// 入口组只展示 Ingress，出口组只展示 Egress；分组顺序沿用后端按 code 排序，不按 ID。
const groups = computed(() => [
  { key: 'ingress', title: '入口节点（Ingress）', nodes: nodes.value.filter((n) => n.type === 1) },
  { key: 'egress', title: '出口节点（Egress）', nodes: nodes.value.filter((n) => n.type !== 1) },
])

function cssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function nodeTypeLabel(type: number): string {
  return String(type) === '1' ? 'Ingress' : 'Egress'
}

function uptimeLabel(seconds: number): string {
  if (seconds <= 0) return '—'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d} 天 ${h} 小时`
  if (h > 0) return `${h} 小时 ${m} 分`
  return `${m} 分`
}

function barWidth(value: number): string {
  return `${Math.min(100, Math.max(0, value))}%`
}

async function loadOverview() {
  loading.value = true
  try {
    nodes.value = await getMonitorOverview()
    // 默认选中第一个入口节点，没有入口时退回第一个节点，便于直接看到时序图。
    if (!selected.value && nodes.value.length > 0) {
      const firstIngress = nodes.value.find((n) => n.type === 1)
      selected.value = (firstIngress ?? nodes.value[0]).code
      await loadHistory()
    }
  } finally {
    loading.value = false
  }
}

async function loadHistory() {
  if (!selected.value) {
    history.value = []
    return
  }
  detailLoading.value = true
  try {
    history.value = await getNodeStatHistory(selected.value)
  } finally {
    detailLoading.value = false
  }
}

async function selectNode(code: string) {
  selected.value = code
  await loadHistory()
}

async function refreshAll() {
  await loadOverview()
  await loadHistory()
}

// 时序图共享坐标轴；返回 any 是为了避免 echarts 复杂联合类型在拼接 yAxis 时过度约束。
function chartFrame(percentAxis: boolean): any {
  const ink3 = cssVar('--ink-3') || '#737373'
  const line = cssVar('--line') || 'rgba(255,255,255,0.1)'
  return {
    grid: { left: 12, right: 18, top: 28, bottom: 8, containLabel: true },
    tooltip: {
      trigger: 'axis',
      backgroundColor: cssVar('--paper-1') || '#0a0a0a',
      borderColor: line,
      textStyle: { color: cssVar('--ink-2') || '#a3a3a3', fontSize: 12 },
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: history.value.map((p) => new Date(p.ts * 1000).toLocaleTimeString('zh-CN', { hour12: false })),
      axisLine: { lineStyle: { color: line } },
      axisLabel: { color: ink3, fontSize: 10 },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
      min: percentAxis ? 0 : undefined,
      max: percentAxis ? 100 : undefined,
      splitLine: { lineStyle: { color: line, type: 'dashed' } },
      axisLabel: { color: ink3, fontSize: 10, formatter: percentAxis ? '{value}%' : undefined },
    },
  }
}

const cpuOption = computed<EChartsOption>(() => {
  void theme.theme
  const accent = cssVar('--accent') || '#ffffff'
  return {
    ...chartFrame(true),
    series: [
      {
        name: 'CPU',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: history.value.map((p) => p.cpu_percent),
        lineStyle: { color: accent, width: 2 },
        itemStyle: { color: accent },
        areaStyle: { opacity: 0.12, color: accent },
      },
    ],
  }
})

const memOption = computed<EChartsOption>(() => {
  void theme.theme
  const ok = cssVar('--ok') || '#4ade80'
  return {
    ...chartFrame(true),
    series: [
      {
        name: '内存',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: history.value.map((p) => p.mem_percent),
        lineStyle: { color: ok, width: 2 },
        itemStyle: { color: ok },
        areaStyle: { opacity: 0.12, color: ok },
      },
    ],
  }
})

const netOption = computed<EChartsOption>(() => {
  void theme.theme
  const accent = cssVar('--accent') || '#ffffff'
  const warn = cssVar('--warn') || '#fbbf24'
  const ink3 = cssVar('--ink-3') || '#737373'
  const frame = chartFrame(false)
  frame.tooltip.valueFormatter = (v: unknown) => `${formatBytes(Number(v))}/s`
  frame.yAxis.axisLabel.formatter = (v: number) => `${formatBytes(v)}/s`
  return {
    ...frame,
    legend: { right: 8, top: 0, textStyle: { color: ink3, fontSize: 11 }, itemWidth: 14, itemHeight: 8 },
    series: [
      {
        name: '入站',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: history.value.map((p) => p.net_in_bps),
        lineStyle: { color: accent, width: 2 },
        itemStyle: { color: accent },
      },
      {
        name: '出站',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: history.value.map((p) => p.net_out_bps),
        lineStyle: { color: warn, width: 2 },
        itemStyle: { color: warn },
      },
    ],
  }
})

onMounted(async () => {
  await loadOverview()
  timer = window.setInterval(() => {
    void loadOverview()
    if (selected.value) void loadHistory()
  }, POLL_MS)
})

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <PaperCard pad="none">
    <div class="flex flex-wrap items-end justify-between gap-3 border-b p-5" style="border-color: var(--line)">
      <div>
        <h2 class="font-paper text-base font-semibold">服务器监控</h2>
      </div>
      <div class="flex items-center gap-2">
        <span class="text-[11px] text-faint">共 {{ nodes.length }} 个节点</span>
        <PaperButton size="sm" @click="refreshAll">
          <RefreshCw class="size-4" :class="(loading || detailLoading) && 'animate-spin'" />
          刷新
        </PaperButton>
      </div>
    </div>

    <div v-if="nodes.length === 0" class="flex items-center justify-center py-16 text-sm text-faint">
      <Activity class="mr-2 size-4" />
      暂无监控数据，等待 Agent 上报节点指标
    </div>

    <div v-else class="flex flex-col gap-5 p-5">
      <section v-for="group in groups" :key="group.key">
        <div class="mb-3 flex items-center gap-2">
          <h3 class="text-sm font-semibold" style="color: var(--ink-1)">{{ group.title }}</h3>
          <span class="text-[11px] text-faint">{{ group.nodes.length }} 个</span>
        </div>
        <p v-if="group.nodes.length === 0" class="py-6 text-center text-[11px] text-faint">该分组暂无节点</p>
        <div v-else class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <button
            v-for="item in group.nodes"
            :key="item.code"
        type="button"
        class="paper-sink rounded-xl p-4 text-left transition-all"
        :style="
          selected === item.code
            ? 'border-color: var(--accent); box-shadow: inset 3px 3px 7px var(--sink-hi), inset -3px -3px 7px var(--sink-lo), 0 0 0 2px var(--accent-soft)'
            : ''
        "
        @click="selectNode(item.code)"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <p class="truncate text-sm font-medium" style="color: var(--ink-1)">{{ item.name }}</p>
            <p class="mt-0.5 font-mono text-[11px] text-faint">{{ item.code }}</p>
          </div>
          <div class="flex shrink-0 flex-col items-end gap-1">
            <PaperBadge :tone="String(item.type) === '1' ? 'default' : 'warn'">{{ nodeTypeLabel(item.type) }}</PaperBadge>
            <PaperBadge :tone="item.status === 'online' ? 'ok' : 'default'" dot>
              {{ item.status === 'online' ? '在线' : '离线' }}
            </PaperBadge>
          </div>
        </div>

        <div v-if="item.metrics" class="mt-3 flex flex-col gap-2">
          <div>
            <div class="flex items-center justify-between text-[11px] text-faint">
              <span class="inline-flex items-center gap-1"><Cpu class="size-3" /> CPU</span>
              <span class="font-mono" style="color: var(--ink-2)">{{ item.metrics.cpu_percent.toFixed(1) }}%</span>
            </div>
            <div class="mt-1 h-1.5 overflow-hidden rounded-full" style="background: var(--paper-3)">
              <div class="h-full rounded-full" :style="{ width: barWidth(item.metrics.cpu_percent), background: 'var(--accent)' }" />
            </div>
          </div>
          <div>
            <div class="flex items-center justify-between text-[11px] text-faint">
              <span class="inline-flex items-center gap-1"><MemoryStick class="size-3" /> 内存</span>
              <span class="font-mono" style="color: var(--ink-2)">
                {{ formatBytes(item.metrics.mem_used) }} / {{ formatBytes(item.metrics.mem_total) }}
              </span>
            </div>
            <div class="mt-1 h-1.5 overflow-hidden rounded-full" style="background: var(--paper-3)">
              <div class="h-full rounded-full" :style="{ width: barWidth(item.metrics.mem_percent), background: 'var(--ok)' }" />
            </div>
          </div>
          <div>
            <div class="flex items-center justify-between text-[11px] text-faint">
              <span class="inline-flex items-center gap-1"><HardDrive class="size-3" /> 磁盘</span>
              <span class="font-mono" style="color: var(--ink-2)">
                {{ formatBytes(item.metrics.disk_used) }} / {{ formatBytes(item.metrics.disk_total) }}
              </span>
            </div>
            <div class="mt-1 h-1.5 overflow-hidden rounded-full" style="background: var(--paper-3)">
              <div class="h-full rounded-full" :style="{ width: barWidth(item.metrics.disk_percent), background: 'var(--warn)' }" />
            </div>
          </div>

          <div class="mt-1 grid grid-cols-2 gap-x-3 gap-y-1 text-[11px] text-faint">
            <span class="inline-flex items-center gap-1">
              <Network class="size-3" />
              入 {{ formatBytes(item.metrics.net_in_bps) }}/s · 出 {{ formatBytes(item.metrics.net_out_bps) }}/s
            </span>
            <span class="inline-flex items-center gap-1">
              <Server class="size-3" />
              负载 {{ item.metrics.load1.toFixed(2) }}
            </span>
            <span class="inline-flex items-center gap-1">
              <Timer class="size-3" />
              运行 {{ uptimeLabel(item.metrics.uptime_sec) }}
            </span>
            <span class="text-right font-mono">{{ formatTime(new Date(item.metrics.ts * 1000).toISOString()) }}</span>
          </div>
        </div>
        <p v-else class="mt-3 py-3 text-center text-[11px] text-faint">等待首次指标上报</p>
      </button>
        </div>
      </section>
    </div>
  </PaperCard>

  <PaperCard pad="none">
    <div class="flex flex-wrap items-end justify-between gap-3 border-b p-5" style="border-color: var(--line)">
      <div>
        <h2 class="font-paper text-base font-semibold">
          节点详情{{ selectedNode ? ` · ${selectedNode.name}` : '' }}
        </h2>
        <p class="mt-1 text-xs text-faint">短时折线图：CPU、内存、网卡入 / 出瞬时流量</p>
      </div>
      <PaperBadge v-if="history.length > 0" tone="ok" dot>{{ history.length }} 个时序点</PaperBadge>
    </div>

    <div v-if="!selectedNode" class="flex items-center justify-center py-16 text-sm text-faint">
      选择上方任一节点查看短时曲线
    </div>
    <div v-else-if="history.length === 0" class="flex items-center justify-center py-16 text-sm text-faint">
      暂无时序数据，请等待 Agent 上报
    </div>
    <div v-else class="grid gap-4 p-5 xl:grid-cols-2">
      <div class="paper-sink rounded-xl p-3">
        <p class="mb-1 text-[11px] text-faint">CPU 使用率</p>
        <PaperEChart :option="cpuOption" height="200px" />
      </div>
      <div class="paper-sink rounded-xl p-3">
        <p class="mb-1 text-[11px] text-faint">内存使用率</p>
        <PaperEChart :option="memOption" height="200px" />
      </div>
      <div class="paper-sink rounded-xl p-3 xl:col-span-2">
        <p class="mb-1 text-[11px] text-faint">网卡瞬时流量（入 / 出）</p>
        <PaperEChart :option="netOption" height="220px" />
      </div>
    </div>
  </PaperCard>
</template>
