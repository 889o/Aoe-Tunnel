<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Package as PackageIcon, Plus, RefreshCw, SquarePen, Trash2 } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { createPackage, deletePackage, listNodes, listPackages, updatePackage } from '@/api'
import type { Node, Package, PackageDuration } from '@/types'
import PaperBadge from '@/components/paper/PaperBadge.vue'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperCard from '@/components/paper/PaperCard.vue'
import PaperDialog from '@/components/paper/PaperDialog.vue'
import PaperInput from '@/components/paper/PaperInput.vue'
import PaperSwitch from '@/components/paper/PaperSwitch.vue'
import { useAuthStore } from '@/stores/auth'
import { formatTime } from '@/lib/utils'

const auth = useAuthStore()
const packages = ref<Package[]>([])
const nodes = ref<Node[]>([])
const loading = ref(false)

const canCreate = computed(() => auth.hasPerm('package:create'))
const canUpdate = computed(() => auth.hasPerm('package:update'))
const canDelete = computed(() => auth.hasPerm('package:delete'))
const showActions = computed(() => canUpdate.value || canDelete.value)

const createOpen = ref(false)
const editOpen = ref(false)
const deleteOpen = ref(false)
const saving = ref(false)
const editing = ref<Package | null>(null)
const deleting = ref<Package | null>(null)

const form = reactive({
  name: '',
  description: '',
  ingress_nodes: [] as string[],
  egress_nodes: [] as string[],
  traffic_limit_gb: '' as string | number,
  rule_limit: '' as string | number,
  periods: [] as PackageDuration[],
  monthly_price: '' as string | number,
  permanent_price: '' as string | number,
  enabled: true,
})

const durationOptions: Array<{ label: string; value: PackageDuration }> = [
  { label: '月付', value: 'monthly' },
  { label: '季付', value: 'quarterly' },
  { label: '半年付', value: 'half_year' },
  { label: '年付', value: 'yearly' },
  { label: '永久', value: 'permanent' },
]

const autoPrices = computed(() => {
  const monthly = Number(form.monthly_price || 0)
  return {
    quarterly: monthly * 3,
    half_year: monthly * 6,
    yearly: monthly * 12,
  }
})

function money(value: number): string {
  return `¥ ${(value ?? 0).toFixed(2)}`
}

function togglePeriod(period: PackageDuration) {
  const idx = form.periods.indexOf(period)
  if (idx >= 0) {
    form.periods.splice(idx, 1)
  } else {
    form.periods.push(period)
  }
}

function pricesSummary(pkg: Package): string {
  return durationOptions
    .filter((opt) => pkg.prices[opt.value] !== undefined)
    .map((opt) => `${opt.label} ${money(pkg.prices[opt.value] ?? 0)}`)
    .join(' · ')
}

// 套餐槽位与节点类型强绑定：入口槽位只列 Ingress，出口槽位只列 Egress。
const ingressNodes = computed(() => nodes.value.filter((n) => String(n.type) === '1'))
const egressNodes = computed(() => nodes.value.filter((n) => String(n.type) === '2'))

function nodeName(code: string): string {
  return nodes.value.find((n) => n.code === code)?.name ?? code
}

function toggle(list: string[], code: string) {
  const idx = list.indexOf(code)
  if (idx >= 0) {
    list.splice(idx, 1)
  } else {
    list.push(code)
  }
}

async function load() {
  loading.value = true
  try {
    packages.value = await listPackages()
  } finally {
    loading.value = false
  }
}

async function loadAll() {
  nodes.value = await listNodes()
  await load()
}

function openCreate() {
  form.name = ''
  form.description = ''
  form.ingress_nodes = []
  form.egress_nodes = []
  form.traffic_limit_gb = ''
  form.rule_limit = ''
  form.periods = ['monthly']
  form.monthly_price = ''
  form.permanent_price = ''
  form.enabled = true
  createOpen.value = true
}

function openEdit(pkg: Package) {
  editing.value = pkg
  form.name = pkg.name
  form.description = pkg.description
  form.ingress_nodes = [...pkg.ingress_nodes]
  form.egress_nodes = [...pkg.egress_nodes]
  form.traffic_limit_gb = pkg.traffic_limit_gb > 0 ? pkg.traffic_limit_gb : ''
  form.rule_limit = pkg.rule_limit > 0 ? pkg.rule_limit : ''
  form.periods = durationOptions
    .map((opt) => opt.value)
    .filter((value) => pkg.prices[value] !== undefined)
  form.monthly_price = pkg.prices.monthly ?? ''
  form.permanent_price = pkg.prices.permanent ?? ''
  form.enabled = pkg.enabled
  editOpen.value = true
}

function round2(value: number): number {
  return Math.round(value * 100) / 100
}

// 月付价格为基准，季付/半年付/年付按月数自动结算；永久单独配置。
function packagePayload() {
  const gb = Number(form.traffic_limit_gb || 0)
  const count = Number(form.rule_limit || 0)
  const monthly = Number(form.monthly_price || 0)
  const permanent = Number(form.permanent_price || 0)
  if (gb < 0 || count < 0 || Number.isNaN(gb) || Number.isNaN(count)) {
    return null
  }
  if (Number.isNaN(monthly) || Number.isNaN(permanent) || monthly < 0 || permanent < 0) {
    return null
  }
  if (form.periods.length === 0) {
    return null
  }
  const prices: Partial<Record<PackageDuration, number>> = {}
  for (const period of form.periods) {
    if (period === 'monthly') {
      prices.monthly = round2(monthly)
    } else if (period === 'quarterly') {
      prices.quarterly = round2(autoPrices.value.quarterly)
    } else if (period === 'half_year') {
      prices.half_year = round2(autoPrices.value.half_year)
    } else if (period === 'yearly') {
      prices.yearly = round2(autoPrices.value.yearly)
    } else {
      prices.permanent = round2(permanent)
    }
  }
  return {
    traffic_limit_gb: gb,
    rule_limit: Math.round(count),
    prices,
  }
}

async function submitCreate() {
  if (!form.name.trim()) {
    toast.error('请填写套餐名称')
    return
  }
  const limits = packagePayload()
  if (!limits) {
    toast.error('请至少选择一个支付周期，流量限制、规则数量与价格必须是非负数字')
    return
  }
  saving.value = true
  try {
    await createPackage({
      name: form.name.trim(),
      description: form.description.trim(),
      ingress_nodes: form.ingress_nodes,
      egress_nodes: form.egress_nodes,
      ...limits,
      enabled: form.enabled,
    })
    toast.success('套餐已创建')
    createOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function submitEdit() {
  if (!editing.value) return
  const limits = packagePayload()
  if (!limits) {
    toast.error('请至少选择一个支付周期，流量限制、规则数量与价格必须是非负数字')
    return
  }
  saving.value = true
  try {
    await updatePackage(editing.value.id, {
      name: form.name.trim(),
      description: form.description.trim(),
      ingress_nodes: form.ingress_nodes,
      egress_nodes: form.egress_nodes,
      ...limits,
      enabled: form.enabled,
    })
    toast.success('套餐已更新，入口规则将按新出口自动重推')
    editOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

function askDelete(pkg: Package) {
  deleting.value = pkg
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!deleting.value) return
  saving.value = true
  try {
    await deletePackage(deleting.value.id)
    toast.success('套餐已删除')
    deleteOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

onMounted(loadAll)
</script>

<template>
  <PaperCard pad="none">
    <div class="flex flex-wrap items-end justify-between gap-3 border-b p-5" style="border-color: var(--line)">
      <div>
        <h2 class="font-paper text-base font-semibold">套餐管理</h2>
      </div>
      <div class="flex items-center gap-2">
        <PaperButton size="sm" @click="load">
          <RefreshCw class="size-4" :class="loading && 'animate-spin'" />
          刷新
        </PaperButton>
        <PaperButton v-if="canCreate" variant="primary" size="sm" @click="openCreate">
          <Plus class="size-4" />
          新建套餐
        </PaperButton>
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="paper-table">
        <thead>
          <tr>
            <th>套餐</th>
            <th>周期价格</th>
            <th>入口节点</th>
            <th>出口节点</th>
            <th>流量限制</th>
            <th>规则数</th>
            <th>状态</th>
            <th>更新时间</th>
            <th v-if="showActions" class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && packages.length === 0">
            <td :colspan="showActions ? 9 : 8" class="py-10 text-center text-faint">加载中…</td>
          </tr>
          <tr v-else-if="packages.length === 0">
            <td :colspan="showActions ? 9 : 8" class="py-10 text-center text-faint">
              <span class="inline-flex items-center gap-2">
                <PackageIcon class="size-4" />
                暂无套餐，先创建套餐并绑定节点
              </span>
            </td>
          </tr>
          <tr v-for="pkg in packages" :key="pkg.id">
            <td>
              <div class="flex flex-col">
                <span class="font-medium" style="color: var(--ink-1)">{{ pkg.name }}</span>
                <span class="mt-0.5 text-[11px] text-faint">#{{ pkg.id }} · {{ pkg.description || '无描述' }}</span>
              </div>
            </td>
            <td class="min-w-48 text-[12px]">{{ pricesSummary(pkg) || '未配置价格' }}</td>
            <td>
              <div class="flex flex-wrap gap-1">
                <PaperBadge v-for="code in pkg.ingress_nodes" :key="code" tone="ok">{{ nodeName(code) }}</PaperBadge>
                <span v-if="pkg.ingress_nodes.length === 0" class="text-[11px] text-faint">未绑定</span>
              </div>
            </td>
            <td>
              <div class="flex flex-wrap gap-1">
                <PaperBadge v-for="code in pkg.egress_nodes" :key="code" tone="warn">{{ nodeName(code) }}</PaperBadge>
                <span v-if="pkg.egress_nodes.length === 0" class="text-[11px] text-faint">未绑定</span>
              </div>
            </td>
            <td class="whitespace-nowrap text-[12px]">
              {{ pkg.traffic_limit_gb > 0 ? `${pkg.traffic_limit_gb} GB` : '不限' }}
            </td>
            <td class="whitespace-nowrap text-[12px]">{{ pkg.rule_limit > 0 ? pkg.rule_limit : '不限' }}</td>
            <td>
              <PaperBadge :tone="pkg.enabled ? 'ok' : 'default'" dot>{{ pkg.enabled ? '启用' : '停用' }}</PaperBadge>
            </td>
            <td class="whitespace-nowrap text-[12px] text-faint">{{ formatTime(pkg.updated_at) }}</td>
            <td v-if="showActions" class="whitespace-nowrap text-right">
              <div class="inline-flex items-center gap-1">
                <PaperButton v-if="canUpdate" size="sm" title="编辑" @click="openEdit(pkg)">
                  <SquarePen class="size-3.5" />
                </PaperButton>
                <PaperButton v-if="canDelete" size="sm" variant="danger" title="删除" @click="askDelete(pkg)">
                  <Trash2 class="size-3.5" />
                </PaperButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="flex items-center justify-between gap-2 border-t px-5 py-3 text-[11px] text-faint" style="border-color: var(--line)">
      <span>共 {{ packages.length }} 个套餐</span>
      <span>一个套餐可一个入口对多个出口，也可多个入口对一个出口</span>
    </div>
  </PaperCard>

  <PaperDialog v-model:open="createOpen" title="新建套餐" description="入口槽位仅列 Ingress 节点，出口槽位仅列 Egress 节点" width="640px">
    <div class="grid gap-4 sm:grid-cols-2">
      <PaperInput v-model="form.name" label="套餐名称" placeholder="标准套餐" />
      <PaperInput v-model="form.description" label="套餐备注" placeholder="可选" />
      <PaperInput
        v-model="form.monthly_price"
        type="number"
        label="月付价格 (元)"
        placeholder="0"
        hint="季付/半年付/年付分别按 3/6/12 个月自动结算"
      />
      <PaperInput
        v-if="form.periods.includes('permanent')"
        v-model="form.permanent_price"
        type="number"
        label="永久价格 (元)"
        placeholder="0"
        hint="永久周期单独配置价格"
      />
      <div class="sm:col-span-2">
        <p class="mb-2 text-xs text-faint">支付周期（可多选）</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="opt in durationOptions"
            :key="opt.value"
            type="button"
            class="rounded-full border px-3 py-1 text-[11px] transition-colors"
            :style="form.periods.includes(opt.value)
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="togglePeriod(opt.value)"
          >
            {{ opt.label }}
          </button>
        </div>
        <p v-if="form.periods.some((p) => p !== 'permanent')" class="mt-2 text-[11px] text-faint">
          自动结算：季付 {{ money(autoPrices.quarterly) }} · 半年付 {{ money(autoPrices.half_year) }} · 年付 {{ money(autoPrices.yearly) }}
        </p>
      </div>
      <PaperInput
        v-model="form.traffic_limit_gb"
        type="number"
        label="流量限制 (GB)"
        placeholder="0"
        hint="0 表示不限"
      />
      <PaperInput
        v-model="form.rule_limit"
        type="number"
        label="可创建规则数量"
        placeholder="0"
        hint="0 表示不限制"
      />
      <div class="sm:col-span-2">
        <p class="mb-2 text-xs text-faint">入口节点（Ingress，可多选）</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="n in ingressNodes"
            :key="n.code"
            type="button"
            class="rounded-full border px-2.5 py-1 text-[11px] transition-colors"
            :style="form.ingress_nodes.includes(n.code)
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="toggle(form.ingress_nodes, n.code)"
          >
            {{ n.name }} ({{ n.code }})
          </button>
          <span v-if="ingressNodes.length === 0" class="text-[11px] text-faint">暂无 Ingress 节点</span>
        </div>
      </div>
      <div class="sm:col-span-2">
        <p class="mb-2 text-xs text-faint">出口节点（Egress，可多选）</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="n in egressNodes"
            :key="n.code"
            type="button"
            class="rounded-full border px-2.5 py-1 text-[11px] transition-colors"
            :style="form.egress_nodes.includes(n.code)
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="toggle(form.egress_nodes, n.code)"
          >
            {{ n.name }} ({{ n.code }})
          </button>
          <span v-if="egressNodes.length === 0" class="text-[11px] text-faint">暂无 Egress 节点</span>
        </div>
      </div>
      <div class="sm:col-span-2">
        <PaperSwitch v-model="form.enabled" label="启用该套餐" />
      </div>
    </div>
    <template #footer>
      <PaperButton @click="createOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="submitCreate">创建</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog v-model:open="editOpen" title="编辑套餐" description="出口集合或启用状态变化会自动重推入口规则" width="640px">
    <div class="grid gap-4 sm:grid-cols-2">
      <PaperInput v-model="form.name" label="套餐名称" />
      <PaperInput v-model="form.description" label="套餐备注" />
      <PaperInput
        v-model="form.monthly_price"
        type="number"
        label="月付价格 (元)"
        placeholder="0"
        hint="季付/半年付/年付分别按 3/6/12 个月自动结算"
      />
      <PaperInput
        v-if="form.periods.includes('permanent')"
        v-model="form.permanent_price"
        type="number"
        label="永久价格 (元)"
        placeholder="0"
        hint="永久周期单独配置价格"
      />
      <div class="sm:col-span-2">
        <p class="mb-2 text-xs text-faint">支付周期（可多选）</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="opt in durationOptions"
            :key="opt.value"
            type="button"
            class="rounded-full border px-3 py-1 text-[11px] transition-colors"
            :style="form.periods.includes(opt.value)
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="togglePeriod(opt.value)"
          >
            {{ opt.label }}
          </button>
        </div>
        <p v-if="form.periods.some((p) => p !== 'permanent')" class="mt-2 text-[11px] text-faint">
          自动结算：季付 {{ money(autoPrices.quarterly) }} · 半年付 {{ money(autoPrices.half_year) }} · 年付 {{ money(autoPrices.yearly) }}
        </p>
      </div>
      <PaperInput
        v-model="form.traffic_limit_gb"
        type="number"
        label="流量限制 (GB)"
        placeholder="0"
        hint="0 表示不限"
      />
      <PaperInput
        v-model="form.rule_limit"
        type="number"
        label="可创建规则数量"
        placeholder="0"
        hint="0 表示不限制"
      />
      <div class="sm:col-span-2">
        <p class="mb-2 text-xs text-faint">入口节点（Ingress，可多选）</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="n in ingressNodes"
            :key="n.code"
            type="button"
            class="rounded-full border px-2.5 py-1 text-[11px] transition-colors"
            :style="form.ingress_nodes.includes(n.code)
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="toggle(form.ingress_nodes, n.code)"
          >
            {{ n.name }} ({{ n.code }})
          </button>
          <span v-if="ingressNodes.length === 0" class="text-[11px] text-faint">暂无 Ingress 节点</span>
        </div>
      </div>
      <div class="sm:col-span-2">
        <p class="mb-2 text-xs text-faint">出口节点（Egress，可多选）</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="n in egressNodes"
            :key="n.code"
            type="button"
            class="rounded-full border px-2.5 py-1 text-[11px] transition-colors"
            :style="form.egress_nodes.includes(n.code)
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="toggle(form.egress_nodes, n.code)"
          >
            {{ n.name }} ({{ n.code }})
          </button>
          <span v-if="egressNodes.length === 0" class="text-[11px] text-faint">暂无 Egress 节点</span>
        </div>
      </div>
      <div class="sm:col-span-2">
        <PaperSwitch v-model="form.enabled" label="启用该套餐" />
      </div>
    </div>
    <template #footer>
      <PaperButton @click="editOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="submitEdit">保存</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog v-model:open="deleteOpen" title="删除套餐" description="仅当套餐下已无端口规则时才允许删除">
    <p class="text-sm text-muted">确认删除套餐「{{ deleting?.name }}」？</p>
    <template #footer>
      <PaperButton @click="deleteOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="confirmDelete">确认删除</PaperButton>
    </template>
  </PaperDialog>
</template>
