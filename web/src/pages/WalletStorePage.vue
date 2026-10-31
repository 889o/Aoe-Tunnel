<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import QRCode from 'qrcode'
import { CheckCircle2, Package as PackageIcon, Plus, ShoppingBag, Wallet } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import {
  createPaymentOrder,
  createRechargeOrder,
  getPaymentMethods,
  getPaymentOrder,
  getRechargeOptions,
  getWallet,
  listPackages,
  purchasePackage,
} from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { Package, PackageDuration, PaymentMethod, PaymentOrderView, WalletView } from '@/types'
import PaperBadge from '@/components/paper/PaperBadge.vue'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperDialog from '@/components/paper/PaperDialog.vue'
import PaperInput from '@/components/paper/PaperInput.vue'
import PaperCard from '@/components/paper/PaperCard.vue'

const auth = useAuthStore()
const wallet = ref<WalletView | null>(null)
const packages = ref<Package[]>([])
const loading = ref(false)
const tab = ref<'wallet' | 'store'>('wallet')

const buyOpen = ref(false)
const buying = ref(false)
const selected = ref<Package | null>(null)
const selectedPeriod = ref<PackageDuration>('monthly')
const methods = ref<PaymentMethod[]>([])
const payType = ref<'balance' | 'wxpay' | 'alipay'>('balance')
const pendingOrder = ref<PaymentOrderView | null>(null)
const buyQrSrc = ref('')
const rechargeOpen = ref(false)
const rechargeBuying = ref(false)
const rechargePresets = ref<number[]>([])
const rechargeAmount = ref<string | number>('')
const rechargePayType = ref<'wxpay' | 'alipay'>('wxpay')
const rechargePending = ref<PaymentOrderView | null>(null)
const rechargeQrSrc = ref('')
let pollTimer: number | null = null

const durationOptions: Array<{ label: string; value: PackageDuration }> = [
  { label: '月付', value: 'monthly' },
  { label: '季付', value: 'quarterly' },
  { label: '半年付', value: 'half_year' },
  { label: '年付', value: 'yearly' },
  { label: '永久', value: 'permanent' },
]

const availablePackages = computed(() => packages.value.filter((p) => p.enabled))
const balance = computed(() => wallet.value?.balance ?? 0)
const rechargeAmountValue = computed(() => {
  const n = typeof rechargeAmount.value === 'number' ? rechargeAmount.value : Number.parseFloat(rechargeAmount.value)
  return Number.isFinite(n) ? n : 0
})

function availablePeriods(pkg: Package): Array<{ label: string; value: PackageDuration }> {
  return durationOptions.filter((opt) => pkg.prices[opt.value] !== undefined)
}

function primaryPeriod(pkg: Package): PackageDuration {
  const periods = availablePeriods(pkg)
  return pkg.prices.monthly !== undefined ? 'monthly' : periods[0]?.value ?? 'monthly'
}

function periodPrice(pkg: Package, period: PackageDuration): number {
  return pkg.prices[period] ?? 0
}

function isPermanentOwned(pkg: Package): boolean {
  return wallet.value?.package_id === pkg.id && wallet.value.package_period === 'permanent'
}

function actionLabel(pkg: Package): string {
  if (isPermanentOwned(pkg)) return '已购买'
  if (wallet.value?.package_id === pkg.id) return '续费'
  return '购买'
}

function currentPeriodLabel(): string {
  const period = wallet.value?.package_period
  if (!period) return '未绑定套餐'
  return `${wallet.value?.package_name || '未绑定套餐'} · ${durationLabel(period)}`
}

function durationLabel(value: PackageDuration): string {
  return durationOptions.find((opt) => opt.value === value)?.label ?? value
}

function money(value: number): string {
  return `¥ ${(value ?? 0).toFixed(2)}`
}

// 支付地址由异步请求返回，直接 window.open 常被浏览器拦截，失败时退回当前页跳转。
function openPayPage(view: PaymentOrderView) {
  if (!view.pay_url) return
  const win = window.open(view.pay_url, '_blank')
  if (!win) {
    window.location.href = view.pay_url
  }
}

// 易支付的 qrcode 字段是二维码内容（支付链接），统一在本地生成二维码图片。
async function prepareQr(view: PaymentOrderView, target: typeof buyQrSrc) {
  const raw = (view.qrcode || view.pay_url || '').trim()
  if (!raw) {
    target.value = ''
    return
  }
  try {
    target.value = await QRCode.toDataURL(raw, { width: 200, margin: 1 })
  } catch {
    target.value = ''
  }
}

async function load() {
  loading.value = true
  try {
    const [w, pkgs, pays, recharge] = await Promise.all([
      getWallet(),
      listPackages(),
      getPaymentMethods(),
      getRechargeOptions().catch(() => ({ presets: [] as number[] })),
    ])
    wallet.value = w
    packages.value = pkgs
    methods.value = pays
    rechargePresets.value = recharge.presets
  } finally {
    loading.value = false
  }
}

function openRecharge() {
  stopPolling()
  rechargePending.value = null
  rechargeQrSrc.value = ''
  rechargeAmount.value = ''
  rechargePayType.value = methods.value[0]?.type ?? 'wxpay'
  rechargeOpen.value = true
}

async function confirmRecharge() {
  const amount = rechargeAmountValue.value
  if (amount < 0.01) {
    toast.error('请输入有效的充值金额')
    return
  }
  if (methods.value.length === 0) {
    toast.error('管理员未开启在线支付')
    return
  }
  rechargeBuying.value = true
  try {
    const res = await createRechargeOrder({ amount, pay_type: rechargePayType.value })
    rechargePending.value = res
    await prepareQr(res, rechargeQrSrc)
    openPayPage(res)
    toast.success('充值订单已创建，请在支付页面完成付款')
    startPolling(res.out_trade_no, () => {
      rechargeOpen.value = false
      rechargePending.value = null
      toast.success('充值成功，余额已到账')
    })
  } finally {
    rechargeBuying.value = false
  }
}

function openBuy(pkg: Package) {
  stopPolling()
  pendingOrder.value = null
  buyQrSrc.value = ''
  selected.value = pkg
  const periods = availablePeriods(pkg)
  selectedPeriod.value = pkg.prices.monthly !== undefined ? 'monthly' : periods[0]?.value ?? 'monthly'
  const price = periodPrice(pkg, selectedPeriod.value)
  payType.value = balance.value >= price ? 'balance' : methods.value[0]?.type ?? 'balance'
  buyOpen.value = true
}

async function confirmBuy() {
  if (!selected.value) return
  const price = periodPrice(selected.value, selectedPeriod.value)
  if (payType.value === 'balance' && balance.value < price) {
    toast.error('余额不足，请选择在线支付')
    return
  }
  buying.value = true
  try {
    if (payType.value === 'balance') {
      await payWithBalance()
      return
    }
    const res = await createPaymentOrder({
      package_id: selected.value.id,
      period: selectedPeriod.value,
      pay_type: payType.value,
    })
    pendingOrder.value = res
    await prepareQr(res, buyQrSrc)
    openPayPage(res)
    toast.success('支付订单已创建，请在支付页面完成付款')
    startPolling(res.out_trade_no, () => {
      toast.success('支付成功，套餐已生效')
      buyOpen.value = false
      tab.value = 'wallet'
      if (auth.user && wallet.value) {
        auth.user.package_id = wallet.value.package_id
        auth.user.package_period = wallet.value.package_period
      }
    })
  } finally {
    buying.value = false
  }
}

async function payWithBalance() {
  if (!selected.value) return
  const res = await purchasePackage(selected.value.id, selectedPeriod.value)
  wallet.value = {
    balance: res.balance,
    package_id: res.package_id,
    package_name: res.package_name,
    package_period: res.package_period,
  }
  if (auth.user) {
    auth.user.balance = res.balance
    auth.user.package_id = res.package_id
    auth.user.package_period = res.package_period
  }
  toast.success(`已购买套餐「${res.package_name}」，周期：${durationLabel(res.package_period)}`)
  buyOpen.value = false
  tab.value = 'wallet'
}

function startPolling(outTradeNo: string, onPaid: () => void) {
  stopPolling()
  let tries = 0
  pollTimer = window.setInterval(async () => {
    tries++
    try {
      const status = await getPaymentOrder(outTradeNo)
      if (status.status === 'paid') {
        stopPolling()
        pendingOrder.value = null
        rechargePending.value = null
        await load()
        onPaid()
        return
      }
      if (status.status === 'failed') {
        stopPolling()
        pendingOrder.value = null
        rechargePending.value = null
        toast.error('支付失败，请重试')
        return
      }
      if (tries >= 150) {
        stopPolling()
        toast.error('支付结果确认超时，请稍后在钱包页刷新查看')
      }
    } catch {
      stopPolling()
    }
  }, 2000)
}

function stopPolling() {
  if (pollTimer !== null) {
    window.clearInterval(pollTimer)
    pollTimer = null
  }
}

function hasMethod(type: 'wxpay' | 'alipay'): boolean {
  return methods.value.some((m) => m.type === type)
}

onMounted(load)
onUnmounted(stopPolling)
</script>

<template>
  <div class="flex flex-col gap-4">
    <PaperCard pad="none">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b p-5" style="border-color: var(--line)">
        <div>
          <h2 class="font-paper text-base font-semibold">钱包商城</h2>
        </div>
        <div class="flex items-center gap-2">
          <PaperButton
            size="sm"
            :variant="tab === 'wallet' ? 'primary' : 'default'"
            @click="tab = 'wallet'"
          >
            <Wallet class="size-4" />
            钱包
          </PaperButton>
          <PaperButton
            size="sm"
            :variant="tab === 'store' ? 'primary' : 'default'"
            @click="tab = 'store'"
          >
            <ShoppingBag class="size-4" />
            购买套餐
          </PaperButton>
        </div>
      </div>

      <div v-if="tab === 'wallet'" class="grid gap-4 p-5 sm:grid-cols-2">
        <div class="paper-sink rounded-2xl p-5">
          <div class="flex items-center gap-2 text-faint">
            <Wallet class="size-4" />
            <span class="text-xs">账户余额</span>
          </div>
          <p class="mt-3 font-paper text-3xl font-semibold" style="color: var(--ink-1)">{{ money(balance) }}</p>
          <p class="mt-2 text-[11px] text-faint">余额可用于购买套餐，也可通过在线支付充值</p>
          <PaperButton class="mt-4" size="sm" variant="primary" @click="openRecharge">
            <Plus class="size-3.5" />
            充值
          </PaperButton>
        </div>

        <div class="paper-sink rounded-2xl p-5">
          <div class="flex items-center gap-2 text-faint">
            <PackageIcon class="size-4" />
            <span class="text-xs">当前套餐</span>
          </div>
          <p class="mt-3 font-paper text-lg font-semibold" style="color: var(--ink-1)">
            {{ currentPeriodLabel() }}
          </p>
          <p class="mt-2 text-[11px] text-faint">
            {{ wallet?.package_id ? `套餐 ID #${wallet.package_id}` : '购买套餐后自动绑定，创建端口规则时生效' }}
          </p>
          <PaperButton class="mt-4" size="sm" variant="primary" @click="tab = 'store'">
            <ShoppingBag class="size-3.5" />
            去购买套餐
          </PaperButton>
        </div>
      </div>

      <div v-else class="p-5">
        <div class="mb-4 flex items-center justify-between">
          <p class="text-xs text-faint">余额 {{ money(balance) }}，购买后自动绑定为当前套餐</p>
          <PaperButton size="sm" @click="load">
            <PackageIcon class="size-3.5" />
            刷新
          </PaperButton>
        </div>

        <div v-if="loading && availablePackages.length === 0" class="py-10 text-center text-faint">加载中…</div>
        <div v-else-if="availablePackages.length === 0" class="paper-sink rounded-xl py-10 text-center">
          <PackageIcon class="mx-auto size-5 text-faint" />
          <p class="mt-2 text-sm text-faint">暂无可购买套餐</p>
        </div>
        <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div v-for="pkg in availablePackages" :key="pkg.id" class="paper-sink flex flex-col rounded-2xl p-5">
            <div class="flex items-start justify-between gap-2">
              <div>
                <p class="font-paper text-sm font-semibold" style="color: var(--ink-1)">{{ pkg.name }}</p>
                <p class="mt-1 line-clamp-2 text-[11px] text-faint">{{ pkg.description || '无描述' }}</p>
              </div>
              <PaperBadge v-if="isPermanentOwned(pkg)" tone="ok" dot>已购买</PaperBadge>
              <PaperBadge v-else-if="wallet?.package_id === pkg.id" tone="warn">当前套餐</PaperBadge>
              <PaperBadge v-else>{{ availablePeriods(pkg).length }} 种周期</PaperBadge>
            </div>

            <div class="mt-4 flex items-end gap-2">
              <span class="font-paper text-2xl font-semibold" style="color: var(--ink-1)">
                {{ money(periodPrice(pkg, primaryPeriod(pkg))) }}
              </span>
              <span class="pb-1 text-[11px] text-faint">/ {{ durationLabel(primaryPeriod(pkg)) }}</span>
            </div>

            <div class="mt-4 flex flex-col gap-1.5 text-[11px] text-faint">
              <span>
                周期价格：{{ availablePeriods(pkg).map((opt) => `${opt.label} ${money(periodPrice(pkg, opt.value))}`).join(' · ') }}
              </span>
              <span>流量限制：{{ pkg.traffic_limit_gb > 0 ? `${pkg.traffic_limit_gb} GB` : '不限' }}</span>
              <span>可创建规则：{{ pkg.rule_limit > 0 ? pkg.rule_limit : '不限' }}</span>
              <span>入口节点：{{ pkg.ingress_nodes.length }} 个 · 出口节点：{{ pkg.egress_nodes.length }} 个</span>
            </div>

            <PaperButton
              class="mt-5"
              variant="primary"
              block
              :disabled="isPermanentOwned(pkg)"
              @click="openBuy(pkg)"
            >
              <CheckCircle2 v-if="isPermanentOwned(pkg)" class="size-4" />
              {{ actionLabel(pkg) }}
            </PaperButton>
          </div>
        </div>
      </div>
    </PaperCard>
  </div>

  <PaperDialog v-model:open="buyOpen" title="确认购买" description="选择支付周期与支付方式，支付成功后自动绑定为当前套餐">
    <div v-if="selected" class="flex flex-col gap-4 text-sm">
      <div class="flex items-center justify-between">
        <span class="text-faint">套餐</span>
        <span>{{ selected.name }}</span>
      </div>
      <div>
        <p class="mb-2 text-xs text-faint">支付周期</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="opt in availablePeriods(selected)"
            :key="opt.value"
            type="button"
            class="rounded-full border px-3 py-1 text-[11px] transition-colors"
            :style="selectedPeriod === opt.value
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="selectedPeriod = opt.value"
          >
            {{ opt.label }} {{ money(periodPrice(selected, opt.value)) }}
          </button>
        </div>
      </div>
      <div>
        <p class="mb-2 text-xs text-faint">支付方式</p>
        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            class="rounded-full border px-3 py-1 text-[11px] transition-colors"
            :disabled="balance < periodPrice(selected, selectedPeriod)"
            :style="payType === 'balance'
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="payType = 'balance'"
          >
            余额支付 {{ money(balance) }}
          </button>
          <button
            v-if="hasMethod('wxpay')"
            type="button"
            class="rounded-full border px-3 py-1 text-[11px] transition-colors"
            :style="payType === 'wxpay'
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="payType = 'wxpay'"
          >
            微信支付
          </button>
          <button
            v-if="hasMethod('alipay')"
            type="button"
            class="rounded-full border px-3 py-1 text-[11px] transition-colors"
            :style="payType === 'alipay'
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="payType = 'alipay'"
          >
            支付宝
          </button>
        </div>
        <div v-if="pendingOrder" class="mt-2 flex flex-col gap-2">
          <p class="text-[11px] text-faint">订单 {{ pendingOrder.out_trade_no }} 待支付，完成支付后会自动生效</p>
          <PaperButton v-if="pendingOrder.pay_url" size="sm" @click="openPayPage(pendingOrder)">
            打开支付页面
          </PaperButton>
          <img v-if="buyQrSrc" :src="buyQrSrc" alt="支付二维码" class="h-40 w-40 self-center rounded-lg" />
        </div>
        <p v-else-if="balance < periodPrice(selected, selectedPeriod) && methods.length === 0" class="mt-2 text-[11px]" style="color: var(--err)">
          余额不足且未开启在线支付，请联系管理员配置易支付
        </p>
      </div>
      <div class="flex items-center justify-between">
        <span class="text-faint">金额</span>
        <span>{{ money(periodPrice(selected, selectedPeriod)) }}</span>
      </div>
      <div v-if="payType === 'balance'" class="flex items-center justify-between">
        <span class="text-faint">购买后余额</span>
        <span>{{ money(balance - periodPrice(selected, selectedPeriod)) }}</span>
      </div>
    </div>
    <template #footer>
      <PaperButton @click="buyOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="buying" @click="confirmBuy">
        {{ payType === 'balance' ? (selected && wallet?.package_id === selected.id ? '确认续费' : '确认购买') : '去支付' }}
      </PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog v-model:open="rechargeOpen" title="余额充值" description="选择快捷金额或自定义金额，支付成功后余额自动到账">
    <div class="flex flex-col gap-4 text-sm">
      <div>
        <p class="mb-2 text-xs text-faint">快捷金额</p>
        <div v-if="rechargePresets.length > 0" class="flex flex-wrap gap-2">
          <button
            v-for="preset in rechargePresets"
            :key="preset"
            type="button"
            class="rounded-full border px-3 py-1 text-[11px] transition-colors"
            :style="rechargeAmountValue === preset
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="rechargeAmount = preset"
          >
            {{ money(preset) }}
          </button>
        </div>
        <p v-else class="text-[11px] text-faint">管理员未配置快捷金额，可直接输入自定义金额</p>
      </div>
      <PaperInput
        v-model="rechargeAmount"
        type="number"
        :min="0.01"
        :step="0.01"
        label="充值金额（元）"
        placeholder="自定义金额"
      />
      <div>
        <p class="mb-2 text-xs text-faint">支付方式</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-if="hasMethod('wxpay')"
            type="button"
            class="rounded-full border px-3 py-1 text-[11px] transition-colors"
            :style="rechargePayType === 'wxpay'
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="rechargePayType = 'wxpay'"
          >
            微信支付
          </button>
          <button
            v-if="hasMethod('alipay')"
            type="button"
            class="rounded-full border px-3 py-1 text-[11px] transition-colors"
            :style="rechargePayType === 'alipay'
              ? 'border-color: var(--accent); background: var(--accent-soft); color: var(--ink-1)'
              : 'border-color: var(--line); color: var(--ink-1)'"
            @click="rechargePayType = 'alipay'"
          >
            支付宝
          </button>
        </div>
        <div v-if="rechargePending" class="mt-2 flex flex-col gap-2">
          <p class="text-[11px] text-faint">订单 {{ rechargePending.out_trade_no }} 待支付，完成支付后余额自动到账</p>
          <PaperButton v-if="rechargePending.pay_url" size="sm" @click="openPayPage(rechargePending)">
            打开支付页面
          </PaperButton>
          <img v-if="rechargeQrSrc" :src="rechargeQrSrc" alt="支付二维码" class="h-40 w-40 self-center rounded-lg" />
        </div>
        <p v-else-if="methods.length === 0" class="mt-2 text-[11px]" style="color: var(--err)">
          管理员未开启在线支付，请联系管理员配置易支付
        </p>
      </div>
      <div class="flex items-center justify-between">
        <span class="text-faint">充值金额</span>
        <span>{{ money(rechargeAmountValue) }}</span>
      </div>
    </div>
    <template #footer>
      <PaperButton @click="rechargeOpen = false">取消</PaperButton>
      <PaperButton variant="primary" :loading="rechargeBuying" :disabled="methods.length === 0" @click="confirmRecharge">
        去支付
      </PaperButton>
    </template>
  </PaperDialog>
</template>
