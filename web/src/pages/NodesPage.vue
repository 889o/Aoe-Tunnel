<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Copy, Download, Gauge, KeyRound, Plus, RefreshCw, SquarePen, Trash2 } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { createNode, deleteNode, getNodeInstall, listNodes, rotateNodeToken, updateNode, updateNodeRate } from '@/api'
import type { InstallBundle, Node } from '@/types'
import PaperBadge from '@/components/paper/PaperBadge.vue'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperCard from '@/components/paper/PaperCard.vue'
import PaperDialog from '@/components/paper/PaperDialog.vue'
import PaperInput from '@/components/paper/PaperInput.vue'
import PaperSelect from '@/components/paper/PaperSelect.vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const nodes = ref<Node[]>([])
const loading = ref(false)

// 写操作列按权限码显隐：创建 / 编辑 / 限速 / 删除。
const canCreate = computed(() => auth.hasPerm('node:create'))
const canUpdate = computed(() => auth.hasPerm('node:update'))
const canDelete = computed(() => auth.hasPerm('node:delete'))
const hasActions = computed(() => canUpdate.value || canDelete.value)

// 入口与出口都可以创建多个；出口同一安装命令可部署多台服务器做负载均衡与故障转移。
const typeOptions = [
  { label: 'Ingress 入口', value: '1' },
  { label: 'Egress 出口', value: '2' },
]

const createOpen = ref(false)
const editOpen = ref(false)
const installOpen = ref(false)
const rotateOpen = ref(false)
const rateOpen = ref(false)
const deleteOpen = ref(false)
const saving = ref(false)
const editing = ref<Node | null>(null)
const rotating = ref<Node | null>(null)
const rating = ref<Node | null>(null)
const deleting = ref<Node | null>(null)
const bundle = ref<InstallBundle | null>(null)

const form = reactive({
  name: '',
  type: '1',
  public_addr: '',
})

const editForm = reactive({
  name: '',
})

const rateForm = reactive({
  rate_up_mbps: '',
  rate_down_mbps: '',
  ratio_up: '',
  ratio_down: '',
})

function isIngress(type: number | string): boolean {
  return String(type) === '1'
}

function nodeTypeLabel(type: number): string {
  return isIngress(type) ? 'Ingress' : 'Egress'
}

function nodeTypeTone(type: number): 'ok' | 'warn' {
  return isIngress(type) ? 'ok' : 'warn'
}

function rateLabel(node: Node): string {
  if (!isIngress(node.type)) return '—'
  const up = node.rate_up_mbps || 0
  const down = node.rate_down_mbps || 0
  if (up <= 0 && down <= 0) return '不限速'
  return `↑${up} / ↓${down} MB/s`
}

// 入口与出口都按节点计费比计费；出口节点不参与限速。
function ratioLabel(node: Node): string {
  return `↑${node.ratio_up || 1} / ↓${node.ratio_down || 1}`
}

async function load() {
  loading.value = true
  try {
    nodes.value = await listNodes()
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.name = ''
  form.type = '2'
  form.public_addr = ''
  createOpen.value = true
}

function openEdit(node: Node) {
  editing.value = node
  editForm.name = node.name
  editOpen.value = true
}

function openRate(node: Node) {
  rating.value = node
  rateForm.rate_up_mbps = node.rate_up_mbps > 0 ? String(node.rate_up_mbps) : ''
  rateForm.rate_down_mbps = node.rate_down_mbps > 0 ? String(node.rate_down_mbps) : ''
  rateForm.ratio_up = String(node.ratio_up || 1)
  rateForm.ratio_down = String(node.ratio_down || 1)
  rateOpen.value = true
}

async function submitCreate() {
  if (!form.name.trim()) {
    toast.error('请填写节点名称')
    return
  }

  saving.value = true
  try {
    const node = await createNode({
      name: form.name.trim(),
      type: Number(form.type),
      // 对接地址只对入口有意义；出口由入口规则选择的地址决定，不填写。
      public_addr: isIngress(form.type) ? form.public_addr.trim() || undefined : undefined,
    })
    createOpen.value = false
    toast.success(`节点已创建，编码 ${node.code}`)
    await load()
    await showInstall(node.code)
  } finally {
    saving.value = false
  }
}

async function submitEdit() {
  if (!editing.value) return
  if (!editForm.name.trim()) {
    toast.error('节点名称不能为空')
    return
  }

  saving.value = true
  try {
    await updateNode(editing.value.code, { name: editForm.name.trim() })
    toast.success('已更新')
    editOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function submitRate() {
  if (!rating.value) return
  const up = Number(rateForm.rate_up_mbps || 0)
  const down = Number(rateForm.rate_down_mbps || 0)
  const ratioUp = Number(rateForm.ratio_up || 1)
  const ratioDown = Number(rateForm.ratio_down || 1)
  if (up < 0 || down < 0 || Number.isNaN(up) || Number.isNaN(down)) {
    toast.error('限速必须是非负数字，0 表示不限速')
    return
  }
  if (ratioUp <= 0 || ratioDown <= 0 || Number.isNaN(ratioUp) || Number.isNaN(ratioDown)) {
    toast.error('计费比必须是大于 0 的数字')
    return
  }

  saving.value = true
  try {
    const payload: { rate_up_mbps?: number; rate_down_mbps?: number; ratio_up: number; ratio_down: number } = {
      ratio_up: ratioUp,
      ratio_down: ratioDown,
    }
    if (isIngress(rating.value.type)) {
      payload.rate_up_mbps = up
      payload.rate_down_mbps = down
    }
    await updateNodeRate(rating.value.code, payload)
    toast.success(isIngress(rating.value.type) ? '节点限速与计费比已更新，入口规则已重新下发' : '出口计费比已更新')
    rateOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deleting.value) return
  saving.value = true
  try {
    await deleteNode(deleting.value.code)
    toast.success('节点已删除')
    deleteOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function showInstall(code: string) {
  try {
    bundle.value = await getNodeInstall(code)
    installOpen.value = true
  } catch {
    toast.error('安装命令生成失败，请稍后重试')
  }
}

async function confirmRotate() {
  if (!rotating.value) return
  saving.value = true
  try {
    await rotateNodeToken(rotating.value.code)
    toast.success('令牌已重置，旧令牌立即失效，请重新下发安装命令')
    rotateOpen.value = false
    await showInstall(rotating.value.code)
  } finally {
    saving.value = false
  }
}

async function copyText(text: string) {
  try {
    if (!navigator.clipboard?.writeText) throw new Error('clipboard unavailable')
    await navigator.clipboard.writeText(text)
    toast.success('已复制到剪贴板')
  } catch {
    // 非 HTTPS 环境没有 Clipboard API，退回隐藏 textarea，execCommand 不受安全上下文限制。
    try {
      const ta = document.createElement('textarea')
      ta.value = text
      ta.style.position = 'fixed'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.select()
      const ok = document.execCommand('copy')
      document.body.removeChild(ta)
      if (!ok) throw new Error('execCommand failed')
      toast.success('已复制到剪贴板')
    } catch {
      toast.error('复制失败，请手动选择文本')
    }
  }
}

function downloadScript(b: InstallBundle) {
  const blob = new Blob([b.script], { type: 'text/x-shellscript' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = b.filename
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(load)
</script>

<template>
  <PaperCard pad="none">
    <div class="flex flex-wrap items-end justify-between gap-3 border-b p-5" style="border-color: var(--line)">
      <div>
        <h2 class="font-paper text-base font-semibold">节点设置</h2>
        <p class="mt-2 text-xs text-faint">
          节点编码自动生成；Ingress 入口负责监听与节点级限速，Egress 出口负责落地
        </p>
      </div>
      <div class="flex items-center gap-2">
        <PaperButton size="sm" @click="load">
          <RefreshCw class="size-4" :class="loading && 'animate-spin'" />
          刷新
        </PaperButton>
        <PaperButton v-if="canCreate" variant="primary" size="sm" @click="openCreate">
          <Plus class="size-4" />
          新建节点
        </PaperButton>
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="paper-table">
        <thead>
          <tr>
            <th>节点</th>
            <th>类型</th>
            <th>状态</th>
            <th>外网 IP</th>
            <th>限速</th>
            <th>计费比</th>
            <th>版本</th>
            <th v-if="hasActions" class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && nodes.length === 0">
            <td :colspan="hasActions ? 8 : 7" class="py-10 text-center text-faint">加载中…</td>
          </tr>
          <tr v-else-if="nodes.length === 0">
            <td :colspan="hasActions ? 8 : 7" class="py-10 text-center text-faint">暂无节点，先创建或启动 Agent 上报</td>
          </tr>
          <tr v-for="node in nodes" :key="node.code">
            <td>
              <div class="flex flex-col">
                <span class="font-medium" style="color: var(--ink-1)">{{ node.name }}</span>
                <span class="mt-0.5 font-mono text-[11px] text-faint">{{ node.code }}</span>
              </div>
            </td>
            <td>
              <PaperBadge :tone="nodeTypeTone(node.type)" dot>{{ nodeTypeLabel(node.type) }}</PaperBadge>
            </td>
            <td>
              <PaperBadge :tone="node.status === 'online' ? 'ok' : 'default'" dot>
                {{ node.status === 'online' ? '在线' : '离线' }}
              </PaperBadge>
            </td>
            <td class="font-mono text-[11px]">{{ node.ip || '—' }}</td>
            <td class="whitespace-nowrap text-[12px]">{{ rateLabel(node) }}</td>
            <td class="whitespace-nowrap font-mono text-[11px]">{{ ratioLabel(node) }}</td>
            <td>{{ node.version || '—' }}</td>
            <td v-if="hasActions" class="whitespace-nowrap text-right">
              <div class="inline-flex items-center gap-1">
                <PaperButton size="sm" title="安装命令" @click="showInstall(node.code)">
                  <Copy class="size-3.5" />
                </PaperButton>
                <PaperButton v-if="canUpdate" size="sm" title="编辑名称" @click="openEdit(node)">
                  <SquarePen class="size-3.5" />
                </PaperButton>
                <PaperButton
                  v-if="canUpdate"
                  size="sm"
                  :title="isIngress(node.type) ? '节点限速与计费比' : '出口计费比'"
                  @click="openRate(node)"
                >
                  <Gauge class="size-3.5" />
                </PaperButton>
                <PaperButton v-if="canUpdate" size="sm" title="重置令牌" @click="rotating = node; rotateOpen = true">
                  <KeyRound class="size-3.5" />
                </PaperButton>
                <PaperButton v-if="canDelete" size="sm" title="删除节点" @click="deleting = node; deleteOpen = true">
                  <Trash2 class="size-3.5" />
                </PaperButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </PaperCard>

  <PaperDialog
    v-model:open="createOpen"
    title="新建节点"
    description="入口与出口均可创建多个；同一节点重复部署时新实例接管、旧实例自动离线，出口可多机部署做负载均衡"
  >
    <div class="grid gap-4 sm:grid-cols-2">
      <PaperInput v-model="form.name" label="节点名称" placeholder="节点名称" />
      <PaperSelect v-model="form.type" label="节点类型" :options="typeOptions" />
      <PaperInput
        v-if="isIngress(form.type)"
        v-model="form.public_addr"
        class="sm:col-span-2"
        label="对接地址（可选）"
        placeholder="自定义内容，如 example.com 或 203.0.113.10"
        hint="留空则不展示；端口规则页的对接地址按此填写值拼接监听端口"
      />
    </div>
    <template #footer>
      <PaperButton @click="createOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="submitCreate">创建</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog v-model:open="editOpen" title="编辑节点" description="仅支持修改节点名称">
    <div class="grid gap-4">
      <PaperInput v-model="editForm.name" label="节点名称" />
    </div>
    <template #footer>
      <PaperButton @click="editOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="submitEdit">保存</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog
    v-model:open="rateOpen"
    :title="rating && isIngress(rating.type) ? '节点限速与计费比' : '出口节点计费比'"
    :description="
      rating && isIngress(rating.type)
        ? '限速与计费比作用于该入口节点上的所有端口规则；限速单位 MB/s，0 表示不限速，计费比必须大于 0'
        : '出口节点只支持计费比，不提供限速功能；计费比必须大于 0'
    "
  >
    <div class="grid gap-4 sm:grid-cols-2">
      <template v-if="rating && isIngress(rating.type)">
        <PaperInput v-model="rateForm.rate_up_mbps" label="上行限速 (MB/s)" placeholder="0" />
        <PaperInput v-model="rateForm.rate_down_mbps" label="下行限速 (MB/s)" placeholder="0" />
      </template>
      <PaperInput v-model="rateForm.ratio_up" label="上行计费比" placeholder="1" />
      <PaperInput v-model="rateForm.ratio_down" label="下行计费比" placeholder="1" />
    </div>
    <template #footer>
      <PaperButton @click="rateOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="submitRate">保存</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog
    v-model:open="installOpen"
    title="一键安装命令"
    description="脚本按节点 code 与令牌动态渲染：自动识别架构、从云存储下载二进制、写入配置并注册 systemd"
    width="720px"
  >
    <div v-if="bundle" class="flex flex-col gap-4">
      <div class="flex flex-wrap items-center gap-2">
        <PaperBadge :tone="bundle.role === 'ingress' ? 'ok' : 'warn'" dot>
          {{ bundle.role === 'ingress' ? 'Ingress 入口' : 'Egress 出口' }}
        </PaperBadge>
        <span class="font-mono text-[11px] text-faint">{{ bundle.code }} · {{ bundle.filename }}</span>
      </div>

      <div v-if="bundle.online_command">
        <p class="mb-1 text-xs text-faint">云存储在线安装（推荐，无需下载脚本）</p>
        <div class="flex items-center gap-2">
          <code class="paper-input flex-1 overflow-x-auto whitespace-nowrap font-mono !text-[12px]">{{ bundle.online_command }}</code>
          <PaperButton size="sm" @click="copyText(bundle.online_command)">
            <Copy class="size-3.5" />
            复制
          </PaperButton>
        </div>
        <p class="mt-1 text-xs text-faint">脚本地址：{{ bundle.install_url }}</p>
      </div>

      <div>
        <p class="mb-1 text-xs text-faint">或下载节点专属脚本，在目标主机保存后执行</p>
        <div class="flex items-center gap-2">
          <code class="paper-input flex-1 font-mono !text-[12px]">{{ bundle.command }}</code>
          <PaperButton size="sm" @click="copyText(bundle.command)">
            <Copy class="size-3.5" />
            复制
          </PaperButton>
        </div>
      </div>

      <div>
        <p class="mb-1 text-xs text-faint">节点令牌（X-Node-Token，仅用于 POST /nodes/:code/online 上线校验）</p>
        <div class="flex items-center gap-2">
          <code class="paper-input flex-1 overflow-x-auto whitespace-nowrap font-mono !text-[12px]">{{ bundle.token }}</code>
          <PaperButton size="sm" @click="copyText(bundle.token)">
            <Copy class="size-3.5" />
          </PaperButton>
        </div>
      </div>

      <div>
        <div class="mb-1 flex items-center justify-between">
          <p class="text-xs text-faint">安装脚本</p>
          <div class="flex items-center gap-2">
            <PaperButton size="sm" @click="copyText(bundle.script)">
              <Copy class="size-3.5" />
              复制脚本
            </PaperButton>
            <PaperButton size="sm" @click="downloadScript(bundle)">
              <Download class="size-3.5" />
              下载
            </PaperButton>
          </div>
        </div>
        <pre
          class="max-h-[320px] overflow-auto rounded-lg border p-3 font-mono text-[11px] leading-relaxed"
          style="border-color: var(--line); background: var(--paper-2); color: var(--ink-2)"
        >{{ bundle.script }}</pre>
      </div>
    </div>
    <template #footer>
      <PaperButton @click="installOpen = false">关闭</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog
    v-model:open="rotateOpen"
    title="重置节点令牌"
    description="旧令牌会立即失效，已部署的 Agent 需重新下发安装命令"
  >
    <p class="text-sm text-muted">
      确认重置节点「{{ rotating?.name }}（{{ rotating?.code }}）」的令牌？旧令牌将立即无法上线。
    </p>
    <template #footer>
      <PaperButton @click="rotateOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="confirmRotate">确认重置</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog
    v-model:open="deleteOpen"
    title="删除节点"
    description="节点上的端口规则会一并删除，套餐中的节点绑定会同步移除"
  >
    <p class="text-sm text-muted">
      确认删除节点「{{ deleting?.name }}（{{ deleting?.code }}）」？该节点作为入口或被引用为出口的端口规则会同时删除，且无法恢复；删除后 Agent 将无法再上线。
    </p>
    <template #footer>
      <PaperButton @click="deleteOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="saving" @click="confirmDelete">确认删除</PaperButton>
    </template>
  </PaperDialog>
</template>
