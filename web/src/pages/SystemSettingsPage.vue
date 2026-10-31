<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { CreditCard, Globe, Save, Share2, SlidersHorizontal, SquarePen } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { getPaymentChannels, getSystemSettings, updatePaymentChannel, updateSystemSettings } from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { PaymentChannel } from '@/types'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperCard from '@/components/paper/PaperCard.vue'
import PaperDialog from '@/components/paper/PaperDialog.vue'
import PaperInput from '@/components/paper/PaperInput.vue'
import PaperSwitch from '@/components/paper/PaperSwitch.vue'

type TabKey = 'site' | 'payment' | 'distribution'

const auth = useAuthStore()
const canUpdate = computed(() => auth.hasPerm('system:update'))

const tabs: { key: TabKey; label: string; icon: unknown }[] = [
  { key: 'site', label: '站点信息', icon: Globe },
  { key: 'payment', label: '支付设置', icon: CreditCard },
  { key: 'distribution', label: '分销设置', icon: Share2 },
]

const active = ref<TabKey>('site')
const loading = ref(false)
const saving = ref(false)

const channels = ref<PaymentChannel[]>([])
const channelsLoading = ref(false)
const configOpen = ref(false)
const configSaving = ref(false)
const editingChannel = ref<PaymentChannel | null>(null)

const configForm = reactive({
  api_url: '',
  pid: '',
  key: '',
  wechat: false,
  alipay: false,
  recharge_presets: '',
})

const form = reactive({
  site_title: '',
  site_subtitle: '',
  announcement: '',
  register_open: false,
  register_review: true,
  distribution_open: false,
  distribution_ratio: '' as string | number,
})

async function load() {
  loading.value = true
  try {
    const cfg = await getSystemSettings()
    form.site_title = cfg.site_title
    form.site_subtitle = cfg.site_subtitle
    form.announcement = cfg.announcement
    form.register_open = cfg.register_open
    form.register_review = cfg.register_review
    form.distribution_open = cfg.distribution_open
    form.distribution_ratio = cfg.distribution_ratio
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!canUpdate.value) {
    toast.error('没有修改系统设置的权限')
    return
  }
  if (!form.site_title.trim()) {
    toast.error('网站标题不能为空')
    return
  }
  const ratio = Number(form.distribution_ratio || 0)
  if (Number.isNaN(ratio) || ratio < 0 || ratio > 100) {
    toast.error('分销比例需在 0-100 之间')
    return
  }
  saving.value = true
  try {
    await updateSystemSettings({
      site_title: form.site_title.trim(),
      site_subtitle: form.site_subtitle.trim(),
      announcement: form.announcement.trim(),
      register_open: form.register_open,
      register_review: form.register_review,
      distribution_open: form.distribution_open,
      distribution_ratio: ratio,
    })
    toast.success('系统设置已保存')
    await load()
  } finally {
    saving.value = false
  }
}

async function loadChannels() {
  channelsLoading.value = true
  try {
    channels.value = await getPaymentChannels()
  } finally {
    channelsLoading.value = false
  }
}

async function toggleChannel(channel: PaymentChannel, enabled: boolean) {
  if (!canUpdate.value) {
    toast.error('没有修改支付设置的权限')
    return
  }
  try {
    await updatePaymentChannel(channel.code, { enabled })
    channel.enabled = enabled
    toast.success(`${channel.name}已${enabled ? '开启' : '关闭'}`)
  } catch {
    // 失败时保持原状态，错误提示由请求层统一处理。
  }
}

function configEnabled(value: string | undefined): boolean {
  return value === '1' || value === 'true' || value === 'on'
}

function openConfig(channel: PaymentChannel) {
  editingChannel.value = channel
  const cfg = channel.config ?? {}
  configForm.api_url = cfg.api_url ?? ''
  configForm.pid = cfg.pid ?? ''
  configForm.key = cfg.key ?? ''
  configForm.wechat = configEnabled(cfg.wechat)
  configForm.alipay = configEnabled(cfg.alipay)
  configForm.recharge_presets = cfg.recharge_presets ?? ''
  configOpen.value = true
}

async function submitConfig() {
  if (!editingChannel.value) return
  configSaving.value = true
  try {
    const updated = await updatePaymentChannel(editingChannel.value.code, {
      config: {
        api_url: configForm.api_url.trim(),
        pid: configForm.pid.trim(),
        key: configForm.key.trim(),
        wechat: configForm.wechat ? '1' : '0',
        alipay: configForm.alipay ? '1' : '0',
        recharge_presets: configForm.recharge_presets.trim(),
      },
    })
    const idx = channels.value.findIndex((ch) => ch.code === updated.code)
    if (idx >= 0) channels.value[idx] = updated
    toast.success('支付配置已保存')
    configOpen.value = false
  } finally {
    configSaving.value = false
  }
}

onMounted(() => {
  void load()
  void loadChannels()
})
</script>

<template>
  <PaperCard pad="none">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b p-5" style="border-color: var(--line)">
      <div>
        <h2 class="font-paper text-base font-semibold">系统设置</h2>
      </div>
      <div class="flex items-center gap-2">
        <PaperButton v-if="canUpdate" variant="primary" size="sm" :loading="saving" @click="save">
          <Save class="size-4" />
          保存
        </PaperButton>
      </div>
    </div>

    <div class="flex flex-wrap gap-2 border-b px-5 py-3" style="border-color: var(--line)">
      <PaperButton
        v-for="tab in tabs"
        :key="tab.key"
        size="sm"
        :variant="active === tab.key ? 'primary' : 'default'"
        @click="active = tab.key"
      >
        <component :is="tab.icon" class="size-3.5" />
        {{ tab.label }}
      </PaperButton>
    </div>

    <div v-if="active === 'site'" class="grid gap-4 p-5 sm:grid-cols-2">
      <PaperInput
        v-model="form.site_title"
        label="网站标题"
        placeholder="Aoe-tunnel"
        :disabled="!canUpdate"
      />
      <PaperInput
        v-model="form.site_subtitle"
        label="副标题"
        placeholder="控制台入口"
        :disabled="!canUpdate"
      />
      <PaperInput
        v-model="form.announcement"
        label="公告"
        placeholder="留空表示不展示公告"
        :disabled="!canUpdate"
        class="sm:col-span-2"
      />
      <div class="paper-sink flex items-center justify-between rounded-xl px-4 py-3">
        <div>
          <p class="text-sm" style="color: var(--ink-1)">是否开启注册</p>
          <p class="mt-0.5 text-[11px] text-faint">关闭后注册页与注册接口均不可用</p>
        </div>
        <PaperSwitch v-model="form.register_open" :disabled="!canUpdate" />
      </div>
      <div class="paper-sink flex items-center justify-between rounded-xl px-4 py-3">
        <div>
          <p class="text-sm" style="color: var(--ink-1)">注册是否需要审核</p>
          <p class="mt-0.5 text-[11px] text-faint">开启后新用户需管理员审核通过才能登录</p>
        </div>
        <PaperSwitch v-model="form.register_review" :disabled="!canUpdate" />
      </div>
    </div>

    <div v-else-if="active === 'payment'" class="p-5">
      <div class="overflow-x-auto">
        <table class="paper-table">
          <thead>
            <tr>
              <th>支付方式</th>
              <th>状态</th>
              <th>配置信息</th>
              <th v-if="canUpdate" class="text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="channelsLoading && channels.length === 0">
              <td :colspan="canUpdate ? 4 : 3" class="py-8 text-center text-faint">加载中…</td>
            </tr>
            <tr v-else-if="channels.length === 0">
              <td :colspan="canUpdate ? 4 : 3" class="py-8 text-center text-faint">暂无支付方式</td>
            </tr>
            <tr v-for="channel in channels" :key="channel.code">
              <td>
                <div class="flex flex-col">
                  <span class="font-medium" style="color: var(--ink-1)">{{ channel.name }}</span>
                  <span class="mt-0.5 font-mono text-[11px] text-faint">{{ channel.code }}</span>
                </div>
              </td>
              <td>
                <div class="flex items-center gap-2">
                  <PaperSwitch
                    :model-value="channel.enabled"
                    :disabled="!canUpdate"
                    @update:model-value="(v: boolean) => toggleChannel(channel, v)"
                  />
                  <span class="text-[12px] text-faint">{{ channel.enabled ? '开启' : '关闭' }}</span>
                </div>
              </td>
              <td class="text-[12px]">
                <span v-if="channel.config?.api_url" class="font-mono text-[11px]">{{ channel.config.api_url }}</span>
                <span v-else class="text-faint">未配置</span>
              </td>
              <td v-if="canUpdate" class="whitespace-nowrap text-right">
                <PaperButton size="sm" title="编辑配置" @click="openConfig(channel)">
                  <SquarePen class="size-3.5" />
                </PaperButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 text-[11px] text-faint">内置支付方式不可删除，仅支持开启/关闭与配置编辑。</p>
    </div>

    <div v-else class="grid gap-4 p-5 sm:grid-cols-2">
      <div class="paper-sink flex items-center justify-between rounded-xl px-4 py-3 sm:col-span-2">
        <div>
          <p class="text-sm" style="color: var(--ink-1)">是否开启分销</p>
          <p class="mt-0.5 text-[11px] text-faint">开启后按分销比例计算推广收益</p>
        </div>
        <PaperSwitch v-model="form.distribution_open" :disabled="!canUpdate" />
      </div>
      <PaperInput
        v-model="form.distribution_ratio"
        type="number"
        label="分销比例 (%)"
        placeholder="0"
        :min="0"
        :max="100"
        :step="0.1"
        :disabled="!canUpdate || !form.distribution_open"
        hint="取值范围 0-100，例如 10 表示 10%"
      />
    </div>

    <div class="border-t px-5 py-3 text-[11px] text-faint" style="border-color: var(--line)">
      <span class="inline-flex items-center gap-1">
        <SlidersHorizontal class="size-3.5" />
        站点标题与副标题会展示在登录、注册页面
      </span>
    </div>
  </PaperCard>

  <PaperDialog
    v-model:open="configOpen"
    :title="`${editingChannel?.name ?? '支付方式'}配置`"
    description="配置易支付网关；异步通知地址由面板根据访问域名自动生成"
    width="560px"
  >
    <div class="grid gap-4">
      <PaperInput v-model="configForm.api_url" label="接口地址" placeholder="https://pay.example.com" />
      <PaperInput v-model="configForm.pid" label="商户 ID" placeholder="10001" autocomplete="off" />
      <PaperInput
        v-model="configForm.key"
        type="password"
        label="商户密钥"
        placeholder="留空沿用已配置密钥"
        autocomplete="new-password"
      />
      <PaperInput
        v-model="configForm.recharge_presets"
        label="快捷充值金额（元）"
        placeholder="10,20,50,100"
        hint="使用逗号分隔；留空使用默认 10/20/50/100/200"
      />
      <div class="paper-sink flex items-center justify-between rounded-xl px-4 py-3">
        <div>
          <p class="text-sm" style="color: var(--ink-1)">微信支付</p>
          <p class="mt-0.5 text-[11px] text-faint">开启后钱包商城可选择微信支付</p>
        </div>
        <PaperSwitch v-model="configForm.wechat" />
      </div>
      <div class="paper-sink flex items-center justify-between rounded-xl px-4 py-3">
        <div>
          <p class="text-sm" style="color: var(--ink-1)">支付宝</p>
          <p class="mt-0.5 text-[11px] text-faint">开启后钱包商城可选择支付宝</p>
        </div>
        <PaperSwitch v-model="configForm.alipay" />
      </div>
    </div>
    <template #footer>
      <PaperButton @click="configOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="configSaving" @click="submitConfig">保存</PaperButton>
    </template>
  </PaperDialog>
</template>
