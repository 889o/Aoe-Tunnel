<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Activity, Plus, RefreshCw, SquarePen, Trash2 } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { createPortRule, deletePortRule, listNodes, listPackages, listPortRules, updatePortRule } from '@/api'
import type { Node, Package, PortRule } from '@/types'
import PaperBadge from '@/components/paper/PaperBadge.vue'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperCard from '@/components/paper/PaperCard.vue'
import PaperDialog from '@/components/paper/PaperDialog.vue'
import PaperInput from '@/components/paper/PaperInput.vue'
import PaperSelect from '@/components/paper/PaperSelect.vue'
import PaperSwitch from '@/components/paper/PaperSwitch.vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const rules = ref<PortRule[]>([])
const nodes = ref<Node[]>([])
const packages = ref<Package[]>([])
const loading = ref(false)

const filterNode = ref('')

const canCreate = computed(() => auth.hasPerm('port:create'))
const canUpdate = computed(() => auth.hasPerm('port:update'))
const canDelete = computed(() => auth.hasPerm('port:delete'))
const showActions = computed(() => canUpdate.value || canDelete.value)

// 套餐由当前登录用户的绑定关系决定，入口只能从该套餐绑定的 Ingress 节点中选择。
const boundPackage = computed(
  () => packages.value.find((p) => p.id === auth.user?.package_id) ?? null,
)
const entryNodes = computed(() => {
  const pkg = boundPackage.value
  if (!pkg) return []
  return nodes.value.filter((n) => n.type === 1 && pkg.ingress_nodes.includes(n.code))
})
const entryNodeOptions = computed(() => entryNodes.value.map((n) => ({ label: `${n.name} (${n.code})`, value: n.code })))
const nodeFilterOptions = computed(() => [{ label: '全部入口', value: '' }, ...entryNodeOptions.value])

// 出口只能从套餐绑定的 Egress 节点中选择。
function exitsForPackage(pkg: Package | null): Node[] {
  if (!pkg) return []
  return nodes.value.filter((n) => n.type === 2 && pkg.egress_nodes.includes(n.code))
}
function exitOptionsFor(pkg: Package | null) {
  return exitsForPackage(pkg).map((n) => ({ label: `${n.name} (${n.code})`, value: n.code }))
}

const egressNodes = computed(() => exitsForPackage(boundPackage.value))
const egressNodeOptions = computed(() => exitOptionsFor(boundPackage.value))

// 协议下拉只选择隧道传输方式；UDP 开关是附加能力，关闭时监听侧仅 TCP。
const transportOptions = [
  { label: 'QUIC', value: 'quic' },
  { label: 'TLS', value: 'tls' },
]

function protocolLabel(protocol: string): string {
  if (protocol === 'both') return 'TCP+UDP'
  return (protocol || 'tcp').toUpperCase()
}

function transportLabel(transport?: string): string {
  return (transport || 'quic').toUpperCase()
}

const createOpen = ref(false)
const editOpen = ref(false)
const deleteOpen = ref(false)
const saving = ref(false)
const editing = ref<PortRule | null>(null)
const deleting = ref<PortRule | null>(null)
const editingPackage = computed(() => packages.value.find((p) => p.id === editing.value?.package_id) ?? null)
const editEgressNodeOptions = computed(() => exitOptionsFor(editingPackage.value))

const form = reactive({
  name: '',
  node: '',
  egress_node: '',
  transport: 'quic',
  udp_enabled: false,
  listen_port: '' as string | number,
  target_addr: '',
})

const editForm = reactive({
  name: '',
  transport: 'quic',
  listen_port: '' as string | number,
  target_addr: '',
  egress_node: '',
  enabled: true,
})

function nodeName(code: string): string {
  return nodes.value.find((n) => n.code === code)?.name ?? code
}

// 对接地址只使用入口节点创建时填写的内容；留空则不展示，不回退 Agent 上报 IP。
function connectAddr(rule: PortRule): string {
  const node = nodes.value.find((n) => n.code === rule.node)
  const host = node?.public_addr?.trim()
  if (!host) return '—'
  return `${host}:${rule.listen_port}`
}

const statusTone = (status: string): 'ok' | 'err' | 'default' => {
  if (status === 'active') return 'ok'
  if (status === 'quota_exhausted') return 'err'
  return 'default'
}

async function load() {
  loading.value = true
  try {
    rules.value = await listPortRules({
      node: filterNode.value || undefined,
      direction: 'entry',
    })
  } finally {
    loading.value = false
  }
}

async function loadAll() {
  const [n, p] = await Promise.all([listNodes(), listPackages()])
  nodes.value = n
  packages.value = p
  await load()
}

function resetForm() {
  form.name = ''
  form.node = entryNodes.value[0]?.code ?? ''
  form.egress_node = egressNodes.value[0]?.code ?? ''
  form.transport = 'quic'
  form.udp_enabled = false
  form.listen_port = ''
  form.target_addr = ''
}

function openCreate() {
  if (!boundPackage.value) {
    toast.error('当前账号未绑定套餐，请联系管理员在用户管理中绑定')
    return
  }
  resetForm()
  createOpen.value = true
}

async function submitCreate() {
  if (!form.name.trim()) {
    toast.error('请填写规则名称')
    return
  }
  if (!form.node) {
    toast.error('请选择入口节点')
    return
  }
  if (!form.egress_node) {
    toast.error('请选择出口节点')
    return
  }
  if (!form.target_addr.trim()) {
    toast.error('请填写目标地址')
    return
  }

  const listenPort = form.listen_port === '' ? 0 : Number(form.listen_port)
  if (listenPort < 0) {
    toast.error('监听端口不合法')
    return
  }

  saving.value = true
  try {
    await createPortRule({
      name: form.name.trim(),
      node: form.node,
      egress_node: form.egress_node,
      protocol: form.udp_enabled ? 'both' : 'tcp',
      transport: form.transport,
      listen_port: listenPort,
      target_addr: form.target_addr.trim(),
    })
    toast.success('规则已创建并下发到所选出口节点')
    createOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

function openEdit(rule: PortRule) {
  editing.value = rule
  editForm.name = rule.name
  editForm.transport = rule.transport || 'quic'
  editForm.listen_port = rule.listen_port
  editForm.target_addr = rule.target_addr
  // 历史规则未记录出口时回退到该套餐第一个出口，保存后即固化为单出口。
  editForm.egress_node = rule.egress_node || editEgressNodeOptions.value[0]?.value || ''
  editForm.enabled = rule.enabled
  editOpen.value = true
}

async function submitEdit() {
  if (!editing.value) return
  if (!editForm.name.trim()) {
    toast.error('请填写规则名称')
    return
  }
  if (!editForm.egress_node) {
    toast.error('请选择出口节点')
    return
  }
  if (!editForm.target_addr.trim()) {
    toast.error('请填写目标地址')
    return
  }
  const listenPort = editForm.listen_port === '' ? 0 : Number(editForm.listen_port)
  if (listenPort < 0) {
    toast.error('监听端口不合法')
    return
  }
  saving.value = true
  try {
    await updatePortRule(editing.value.id, {
      name: editForm.name.trim(),
      transport: editForm.transport,
      listen_port: listenPort,
      target_addr: editForm.target_addr.trim(),
      egress_node: editForm.egress_node,
      enabled: editForm.enabled,
    })
    toast.success('规则已更新并下发')
    editOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

function askDelete(rule: PortRule) {
  deleting.value = rule
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!deleting.value) return
  saving.value = true
  try {
    await deletePortRule(deleting.value.id)
    toast.success('规则已删除，监听已关闭')
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
        <h2 class="font-paper text-base font-semibold">转发规则</h2>
        <p v-if="boundPackage" class="mt-1 text-[11px] text-faint">
          当前套餐：{{ boundPackage.name }} · 可创建 {{ boundPackage.rule_limit > 0 ? boundPackage.rule_limit : '不限量' }}
        </p>
        <p v-else class="mt-1 text-[11px]" style="color: var(--err)">当前账号未绑定套餐，无法创建规则</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <div class="w-44">
          <PaperSelect v-model="filterNode" :options="nodeFilterOptions" @update:model-value="load" />
        </div>
        <PaperButton size="sm" @click="load">
          <RefreshCw class="size-4" :class="loading && 'animate-spin'" />
          刷新
        </PaperButton>
        <PaperButton v-if="canCreate" variant="primary" size="sm" @click="openCreate">
          <Plus class="size-4" />
          新建规则
        </PaperButton>
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="paper-table">
        <thead>
          <tr>
            <th>规则名称</th>
            <th>入口节点名称</th>
            <th>对接地址</th>
            <th>状态</th>
            <th v-if="showActions" class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && rules.length === 0">
            <td :colspan="showActions ? 5 : 4" class="py-10 text-center text-faint">加载中…</td>
          </tr>
          <tr v-else-if="rules.length === 0">
            <td :colspan="showActions ? 5 : 4" class="py-10 text-center text-faint">暂无端口规则</td>
          </tr>
          <tr v-for="rule in rules" :key="rule.id">
            <td>
              <div class="flex flex-col">
                <span class="font-medium" style="color: var(--ink-1)">{{ rule.name }}</span>
                <span class="mt-0.5 font-mono text-[11px] text-faint">#{{ rule.id }} · rev {{ rule.revision }}</span>
              </div>
            </td>
            <td class="text-[12px]">
              <div class="flex flex-col">
                <span>{{ nodeName(rule.node) }}</span>
                <span class="font-mono text-[11px] text-faint">{{ rule.node }}</span>
              </div>
            </td>
            <td class="font-mono text-[12px] whitespace-nowrap">
              <span>{{ connectAddr(rule) }}</span>
              <span class="ml-1 text-[11px] text-faint">{{ transportLabel(rule.transport) }} · {{ protocolLabel(rule.protocol) }}</span>
            </td>
            <td>
              <div class="flex flex-col gap-1">
                <PaperBadge :tone="statusTone(rule.status)" dot>{{ rule.status }}</PaperBadge>
                <PaperBadge v-if="!rule.enabled">已停用</PaperBadge>
              </div>
            </td>
            <td v-if="showActions" class="text-right whitespace-nowrap">
              <div class="inline-flex items-center gap-1">
                <PaperButton v-if="canUpdate" size="sm" title="编辑" @click="openEdit(rule)">
                  <SquarePen class="size-3.5" />
                </PaperButton>
                <PaperButton v-if="canDelete" size="sm" variant="danger" title="删除" @click="askDelete(rule)">
                  <Trash2 class="size-3.5" />
                </PaperButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="flex items-center justify-between gap-2 border-t px-5 py-3 text-[11px] text-faint" style="border-color: var(--line)">
      <span>共 {{ rules.length }} 条规则</span>
      <span class="inline-flex items-center gap-1">
        <Activity class="size-3.5" />
        计费比在节点配置维护，出口节点不参与限速，套餐流量超限时自动断流
      </span>
    </div>
  </PaperCard>

  <PaperDialog
    v-model:open="createOpen"
    title="新建端口规则"
    description="选择隧道协议（QUIC/TLS），UDP 开关用于附加 UDP 监听；出口从套餐绑定的 Egress 节点中选择"
    width="640px"
  >
    <div class="grid gap-4 sm:grid-cols-2">
      <PaperInput v-model="form.name" label="规则名称" placeholder="tcp-demo" />
      <PaperSelect v-model="form.node" label="选择入口" :options="entryNodeOptions" placeholder="选择入口节点" />
      <PaperSelect
        v-model="form.egress_node"
        label="选择出口"
        :options="egressNodeOptions"
        placeholder="选择出口节点"
      />
      <PaperSelect v-model="form.transport" label="选择协议" :options="transportOptions" />
      <PaperSwitch v-model="form.udp_enabled" label="开启 UDP（关闭则仅 TCP）" />
      <PaperInput
        v-model="form.listen_port"
        type="number"
        label="监听端口"
        placeholder="留空随机分配"
        :min="0"
        :max="65535"
        hint="留空或 0 时由后端自动分配随机端口"
      />
      <PaperInput
        v-model="form.target_addr"
        class="sm:col-span-2"
        label="目标地址"
        placeholder="127.0.0.1:8080"
        hint="出口节点实际 Dial 的目标地址，必填"
      />
      <p class="sm:col-span-2 text-[11px] text-faint">
        选择协议决定入口到出口的隧道链路：QUIC 与 TLS 固定生效，不做跨协议自动降级；后续可编辑规则修改协议。
      </p>
      <p class="sm:col-span-2 text-[11px] text-faint">
        UDP 为附加能力：开启后同一监听端口同时接受 TCP 与 UDP；关闭时全部流量走 TCP。
      </p>
      <p class="sm:col-span-2 text-[11px] text-faint">
        限速与计费比在入口节点的「节点配置」中修改。
      </p>
    </div>
    <template #footer>
      <PaperButton @click="createOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="submitCreate">创建并下发</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog
    v-model:open="editOpen"
    title="编辑端口规则"
    description="入口与 UDP 监听不可修改；可调整隧道协议、目标地址、出口节点与监听端口"
  >
    <div class="grid gap-4">
      <PaperSelect v-model="editForm.transport" label="选择协议" :options="transportOptions" />
      <PaperInput v-model="editForm.name" label="规则名称" />
      <PaperSelect
        v-model="editForm.egress_node"
        label="选择出口"
        :options="editEgressNodeOptions"
        placeholder="选择出口节点"
      />
      <PaperInput v-model="editForm.target_addr" label="目标地址" placeholder="127.0.0.1:8080" />
      <PaperInput
        v-model="editForm.listen_port"
        type="number"
        label="监听端口"
        :min="0"
        :max="65535"
        hint="留空或 0 时重新自动分配随机端口"
      />
      <div class="sm:col-span-2">
        <PaperSwitch v-model="editForm.enabled" label="启用该规则" />
      </div>
    </div>
    <template #footer>
      <PaperButton @click="editOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="submitEdit">保存并下发</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog v-model:open="deleteOpen" title="删除端口规则" description="删除后 Agent 会立即关闭该监听并清理本地缓存">
    <p class="text-sm text-muted">
      确认删除规则「{{ deleting?.name }}」？监听 {{ protocolLabel(deleting?.protocol ?? '') }}/{{ deleting?.listen_port }} 将停止服务，所选出口的合成规则也会同步清理。
    </p>
    <template #footer>
      <PaperButton @click="deleteOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="confirmDelete">确认删除</PaperButton>
    </template>
  </PaperDialog>
</template>
