<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { EChartsOption } from 'echarts'
import {
  ArrowDownToLine,
  ArrowUpFromLine,
  Boxes,
  Gauge,
  Package as PackageIcon,
  RefreshCw,
} from 'lucide-vue-next'
import { getDashboard } from '@/api'
import type { DashboardSummary } from '@/types'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperCard from '@/components/paper/PaperCard.vue'
import PaperEChart from '@/components/paper/PaperEChart.vue'
import { useThemeStore } from '@/stores/theme'
import { formatBytes } from '@/lib/utils'

const theme = useThemeStore()
const summary = ref<DashboardSummary | null>(null)
const loading = ref(false)

const periodLabels: Record<string, string> = {
  monthly: '月付',
  quarterly: '季付',
  half_year: '半年付',
  yearly: '年付',
  permanent: '永久',
}

const periodLabel = computed(() => {
  const period = summary.value?.package.period
  if (!period) return ''
  return periodLabels[period] ?? period
})

const ruleRemainingLabel = computed(() => {
  const pkg = summary.value?.package
  if (!pkg || pkg.rule_remaining < 0) return '不限'
  return String(pkg.rule_remaining)
})

const ruleDetail = computed(() => {
  const pkg = summary.value?.package
  if (!pkg?.name) return '未绑定套餐'
  if (pkg.rule_limit <= 0) return `已用 ${pkg.rule_used} 条 · 不限总数`
  return `已用 ${pkg.rule_used} / ${pkg.rule_limit} 条`
})

const trafficRemainingLabel = computed(() => {
  const pkg = summary.value?.package
  if (!pkg || pkg.traffic_remaining_bytes < 0) return '不限'
  return formatBytes(pkg.traffic_remaining_bytes)
})

const trafficDetail = computed(() => {
  const pkg = summary.value?.package
  if (!pkg?.name) return '未绑定套餐'
  if (pkg.traffic_limit_bytes <= 0) return `已用 ${formatBytes(pkg.traffic_used_bytes)} · 不限总量`
  return `已用 ${formatBytes(pkg.traffic_used_bytes)} / ${pkg.traffic_limit_gb} GB`
})

function cssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

const chartOption = computed<EChartsOption>(() => {
  void theme.theme
  const ink2 = cssVar('--ink-2') || '#bcb4a4'
  const ink3 = cssVar('--ink-3') || '#857e71'
  const line = cssVar('--line') || 'rgba(255,255,255,0.08)'
  const paper1 = cssVar('--paper-1') || '#232220'
  const accent = cssVar('--accent') || '#c8a97e'
  const ok = cssVar('--ok') || '#8aa878'
  const series = summary.value?.series ?? []
  const labels = series.map((s) => new Date(s.bucket).toLocaleString('zh-CN', { hour12: false }))

  return {
    backgroundColor: 'transparent',
    grid: { left: 12, right: 18, top: 34, bottom: 8, containLabel: true },
    tooltip: {
      trigger: 'axis',
      backgroundColor: paper1,
      borderColor: line,
      textStyle: { color: ink2, fontSize: 12 },
      valueFormatter: (v: unknown) => formatBytes(Number(v)),
    },
    legend: { right: 8, top: 0, textStyle: { color: ink3, fontSize: 11 }, itemWidth: 14, itemHeight: 8 },
    xAxis: {
      type: 'category',
      data: labels,
      boundaryGap: false,
      axisLine: { lineStyle: { color: line } },
      axisLabel: { color: ink3, fontSize: 10 },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: line, type: 'dashed' } },
      axisLabel: { color: ink3, fontSize: 10, formatter: (v: number) => formatBytes(v) },
    },
    series: [
      {
        name: '上行',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: series.map((s) => s.up_bytes),
        lineStyle: { color: accent, width: 2 },
        itemStyle: { color: accent },
      },
      {
        name: '下行',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: series.map((s) => s.down_bytes),
        lineStyle: { color: ok, width: 2 },
        itemStyle: { color: ok },
      },
    ],
  }
})

async function load() {
  loading.value = true
  try {
    summary.value = await getDashboard()
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-2">
    <h1 class="font-paper text-base font-semibold">概览</h1>
    <PaperButton size="sm" :loading="loading" @click="load">
      <RefreshCw class="size-4" :class="loading && 'animate-spin'" />
      刷新
    </PaperButton>
  </div>

  <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
    <PaperCard pad="sm">
      <div class="flex items-center justify-between">
        <span class="text-xs text-faint">当前套餐</span>
        <PackageIcon class="size-4" style="color: var(--accent)" />
      </div>
      <p class="mt-2 truncate font-paper text-2xl font-semibold">
        {{ summary?.package.name || '未绑定' }}
      </p>
      <p class="mt-1 text-[11px] text-faint">{{ periodLabel || '暂无计费周期' }}</p>
    </PaperCard>

    <PaperCard pad="sm">
      <div class="flex items-center justify-between">
        <span class="text-xs text-faint">剩余端口规则</span>
        <Boxes class="size-4" style="color: var(--ok)" />
      </div>
      <p class="mt-2 font-paper text-2xl font-semibold">{{ ruleRemainingLabel }}</p>
      <p class="mt-1 text-[11px] text-faint">{{ ruleDetail }}</p>
    </PaperCard>

    <PaperCard pad="sm">
      <div class="flex items-center justify-between">
        <span class="text-xs text-faint">剩余流量</span>
        <Gauge class="size-4" style="color: var(--warn)" />
      </div>
      <p class="mt-2 font-paper text-2xl font-semibold">{{ trafficRemainingLabel }}</p>
      <p class="mt-1 text-[11px] text-faint">{{ trafficDetail }}</p>
    </PaperCard>

    <PaperCard pad="sm">
      <div class="flex items-center justify-between">
        <span class="text-xs text-faint">24h 计费流量</span>
        <ArrowUpFromLine class="size-4" style="color: var(--ink-3)" />
      </div>
      <p class="mt-2 font-paper text-2xl font-semibold">{{ formatBytes(summary?.traffic_24h.bill_bytes ?? 0) }}</p>
      <p class="mt-1 text-[11px] text-faint">
        <ArrowUpFromLine class="mr-1 inline size-3.5" />{{ formatBytes(summary?.traffic_24h.up_bytes ?? 0) }}
        <ArrowDownToLine class="ml-2 mr-1 inline size-3.5" />{{ formatBytes(summary?.traffic_24h.down_bytes ?? 0) }}
      </p>
    </PaperCard>
  </div>

  <PaperCard pad="sm">
    <div class="mb-1">
      <h2 class="font-paper text-base font-semibold">24 小时流量趋势</h2>
      <p class="mt-0.5 text-xs text-faint">当前账号规则按小时聚合的上下行字节数</p>
    </div>
    <PaperEChart v-if="summary && summary.series.length > 0" :option="chartOption" height="300px" />
    <div v-else class="flex h-[300px] items-center justify-center text-sm text-faint">暂无流量数据</div>
  </PaperCard>
</template>
