<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Plus, RefreshCw, SquarePen, Trash2, UserX } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { createUser, deleteUser, listPackages, listUsers, updateUser } from '@/api'
import type { Package, UserView } from '@/types'
import PaperBadge from '@/components/paper/PaperBadge.vue'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperCard from '@/components/paper/PaperCard.vue'
import PaperDialog from '@/components/paper/PaperDialog.vue'
import PaperInput from '@/components/paper/PaperInput.vue'
import PaperSelect from '@/components/paper/PaperSelect.vue'
import PaperSwitch from '@/components/paper/PaperSwitch.vue'
import { useAuthStore } from '@/stores/auth'
import { formatTime } from '@/lib/utils'

const auth = useAuthStore()
const users = ref<UserView[]>([])
const packages = ref<Package[]>([])
const loading = ref(false)

const createOpen = ref(false)
const editOpen = ref(false)
const deleteOpen = ref(false)
const saving = ref(false)
const editing = ref<UserView | null>(null)
const deleting = ref<UserView | null>(null)

const isAdminActor = computed(() => ['admin', 'super_admin'].includes(auth.role?.code ?? ''))
const canEditRole = computed(() => {
  if (!isAdminActor.value || !editing.value) return false
  if (editing.value.id === auth.user?.id) return false
  return editing.value.role_code !== 'super_admin'
})
const packageOptions = computed(() => [
  { label: '不绑定套餐', value: '' },
  ...packages.value.map((p) => ({ label: `${p.name} (#${p.id})`, value: String(p.id) })),
])
const statusOptions = [
  { label: '启用', value: 'active' },
  { label: '待审核', value: 'pending' },
  { label: '禁用', value: 'disabled' },
]

function statusLabel(status: string): string {
  if (status === 'active') return '启用'
  if (status === 'pending') return '待审核'
  return '禁用'
}

function statusTone(status: string): 'ok' | 'warn' | 'err' {
  if (status === 'active') return 'ok'
  if (status === 'pending') return 'warn'
  return 'err'
}

function balanceLabel(balance: number): string {
  return `¥ ${(balance ?? 0).toFixed(2)}`
}

const form = reactive({
  username: '',
  password: '',
  is_admin: false,
  package_id: '',
  status: 'active',
})
const editForm = reactive({ is_admin: false, package_id: '', status: 'active', password: '' })

function packageName(id: number | null): string {
  if (!id) return '未绑定'
  return packages.value.find((p) => p.id === id)?.name ?? `#${id}`
}

async function load() {
  loading.value = true
  try {
    users.value = await listUsers()
    packages.value = await listPackages()
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.username = ''
  form.password = ''
  form.is_admin = false
  form.package_id = ''
  form.status = 'active'
  createOpen.value = true
}

function openEdit(user: UserView) {
  editing.value = user
  editForm.is_admin = user.role_code === 'admin' || user.role_code === 'super_admin'
  editForm.package_id = user.package_id ? String(user.package_id) : ''
  editForm.status = user.status
  editForm.password = ''
  editOpen.value = true
}

async function submitCreate() {
  if (!form.username.trim() || !form.password) {
    toast.error('请填写用户名与密码')
    return
  }
  saving.value = true
  try {
    await createUser({
      username: form.username.trim(),
      password: form.password,
      role_code: form.is_admin ? 'admin' : 'user',
      package_id: form.package_id ? Number(form.package_id) : 0,
      status: form.status,
    })
    toast.success('用户已创建')
    createOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function submitEdit() {
  if (!editing.value) return
  const input: { role_code?: string; package_id?: number; status?: string; password?: string } = {
    package_id: editForm.package_id ? Number(editForm.package_id) : 0,
    status: editForm.status,
  }
  // 角色只区分管理员与用户，仅管理员可调整，且不能修改自己的角色。
  if (canEditRole.value) {
    const targetRole = editForm.is_admin ? 'admin' : 'user'
    const currentRole = editing.value.role_code === 'super_admin' ? 'super_admin' : editing.value.role_code
    if (targetRole !== currentRole) {
      input.role_code = targetRole
    }
  }
  if (editForm.password.trim()) {
    input.password = editForm.password.trim()
  }
  saving.value = true
  try {
    await updateUser(editing.value.id, input)
    toast.success('用户已更新')
    editOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

function askDelete(user: UserView) {
  deleting.value = user
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!deleting.value) return
  saving.value = true
  try {
    await deleteUser(deleting.value.id)
    toast.success('用户已删除，其会话已同步失效')
    deleteOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <PaperCard pad="none">
    <div class="flex items-center justify-between gap-3 border-b p-5" style="border-color: var(--line)">
      <div>
        <h2 class="font-paper text-base font-semibold">用户管理</h2>
      </div>
      <div class="flex items-center gap-2">
        <PaperButton size="sm" @click="load">
          <RefreshCw class="size-4" :class="loading && 'animate-spin'" />
          刷新
        </PaperButton>
        <PaperButton v-permission="'user:create'" variant="primary" size="sm" @click="openCreate">
          <Plus class="size-4" />
          新建用户
        </PaperButton>
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="paper-table">
        <thead>
          <tr>
            <th>用户名</th>
            <th>余额</th>
            <th>用户套餐</th>
            <th>状态</th>
            <th>创建时间</th>
            <th>最后登录时间</th>
            <th class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && users.length === 0">
            <td colspan="7" class="py-10 text-center text-faint">加载中…</td>
          </tr>
          <tr v-else-if="users.length === 0">
            <td colspan="7" class="py-10 text-center text-faint">暂无用户</td>
          </tr>
          <tr v-for="user in users" :key="user.id">
            <td>
              <div class="flex flex-col">
                <span class="font-medium" style="color: var(--ink-1)">
                  {{ user.username }}
                  <span v-if="user.id === auth.user?.id" class="ml-1 text-[11px] text-faint">(我)</span>
                </span>
                <span class="font-mono text-[11px] text-faint">#{{ user.id }}</span>
              </div>
            </td>
            <td class="font-mono text-[12px] whitespace-nowrap">{{ balanceLabel(user.balance) }}</td>
            <td class="text-[12px]">{{ packageName(user.package_id) }}</td>
            <td>
              <PaperBadge :tone="statusTone(user.status)" dot>{{ statusLabel(user.status) }}</PaperBadge>
            </td>
            <td class="text-[12px] whitespace-nowrap">{{ formatTime(user.created_at) }}</td>
            <td class="text-[12px] whitespace-nowrap">{{ formatTime(user.last_login_at) }}</td>
            <td class="text-right whitespace-nowrap">
              <div class="inline-flex items-center gap-1">
                <PaperButton v-permission="'user:update'" size="sm" title="编辑" @click="openEdit(user)">
                  <SquarePen class="size-3.5" />
                </PaperButton>
                <PaperButton
                  v-permission="'user:delete'"
                  size="sm"
                  variant="danger"
                  title="删除"
                  :disabled="user.id === auth.user?.id"
                  @click="askDelete(user)"
                >
                  <Trash2 class="size-3.5" />
                </PaperButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="border-t px-5 py-3 text-[11px] text-faint" style="border-color: var(--line)">
      共 {{ users.length }} 个用户 · 非超级管理员不得操作 super_admin 账号
    </div>
  </PaperCard>

  <PaperDialog v-model:open="createOpen" title="新建用户" description="创建后即可用账号密码登录面板" width="560px">
    <div class="grid gap-4 sm:grid-cols-2">
      <PaperInput v-model="form.username" label="用户名" placeholder="alice" autocomplete="off" />
      <PaperInput v-model="form.password" type="password" label="初始密码" placeholder="至少 6 位" autocomplete="new-password" />
      <PaperSwitch v-model="form.is_admin" label="设为管理员" />
      <PaperSelect v-model="form.package_id" label="绑定套餐" :options="packageOptions" />
      <PaperSelect v-model="form.status" label="状态" :options="statusOptions" />
    </div>
    <template #footer>
      <PaperButton size="sm" @click="createOpen = false">取消</PaperButton>
      <PaperButton variant="primary" size="sm" :loading="saving" @click="submitCreate">创建</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog v-model:open="editOpen" title="编辑用户" width="560px">
    <div class="grid gap-4 sm:grid-cols-2">
      <PaperSwitch v-model="editForm.is_admin" label="管理员" :disabled="!canEditRole" />
      <PaperSelect v-model="editForm.package_id" label="绑定套餐" :options="packageOptions" />
      <PaperSelect v-model="editForm.status" label="状态" :options="statusOptions" :disabled="editing?.id === auth.user?.id" />
      <PaperInput v-model="editForm.password" type="password" label="重置密码" placeholder="留空不修改" autocomplete="new-password" />
    </div>
    <p class="mt-3 text-[11px] text-faint">改密或禁用后，该用户已登录的会话会全部失效。</p>
    <template #footer>
      <PaperButton size="sm" @click="editOpen = false">取消</PaperButton>
      <PaperButton variant="primary" size="sm" :loading="saving" @click="submitEdit">保存</PaperButton>
    </template>
  </PaperDialog>

  <PaperDialog v-model:open="deleteOpen" title="删除用户" width="460px">
    <p class="text-sm text-muted">
      确认删除用户
      <span class="font-medium" style="color: var(--ink-1)">{{ deleting?.username }}</span>
      ？其会话会被一并失效，此操作不可恢复。
    </p>
    <template #footer>
      <PaperButton size="sm" @click="deleteOpen = false">取消</PaperButton>
      <PaperButton variant="primary" size="sm" :loading="saving" @click="confirmDelete">
        <UserX class="size-4" />
        确认删除
      </PaperButton>
    </template>
  </PaperDialog>
</template>
