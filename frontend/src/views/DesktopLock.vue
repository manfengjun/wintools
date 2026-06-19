<script setup>
import { ref, onMounted } from 'vue'
import IconLock from '../components/icons/IconLock.vue'
import { useT } from '../locale.js'
import { Status, Lock, Unlock, Backup, Restore, VerifyPassword, ListBackups, DeleteBackup, GetBackupIcons } from '../../wailsjs/go/desktoplock/API'

const locked = ref(false)
const backupCount = ref(0)
const desktopCount = ref(0)
const missing = ref([])
const statusText = ref('')
const showPwdDialog = ref(false)
const pwdInput = ref('')
const pwdError = ref('')
const toast = ref('')
const toastType = ref('info')
const backupList = ref([])
const showBackupList = ref(false)
const confirmDeleteTarget = ref('')
const t = useT()

function showToast(msg, type = 'info') {
  toast.value = msg
  toastType.value = type
  setTimeout(() => { toast.value = '' }, 3000)
}

async function refreshStatus() {
  try {
    const s = await Status()
    locked.value = s.locked
    backupCount.value = s.backup_count
    desktopCount.value = s.desktop_count
    missing.value = s.missing || []
    statusText.value = s.locked ? t('desktopLock.statusLocked') : t('desktopLock.statusUnlocked')
  } catch {
    statusText.value = t('desktopLock.fetchFailed')
  }
}

async function doLock() {
  await Lock()
  showToast(t('desktopLock.lockedToast'), 'success')
  refreshStatus()
}

async function doUnlock() {
  await Unlock()
  showToast(t('desktopLock.unlockedToast'), 'success')
  refreshStatus()
}

async function doBackup() {
  const r = await Backup()
  showToast(t('desktopLock.backupToast').replace('{n}', r.ok), 'success')
  refreshStatus()
}

async function doRestore() {
  const r = await Restore()
  showToast(t('desktopLock.restoreToast').replace('{n}', r.restored), 'success')
  refreshStatus()
}

async function listBackups() {
  showBackupList.value = !showBackupList.value
  if (showBackupList.value) {
    try {
      const items = await ListBackups()
      if (!items || !Array.isArray(items)) {
        backupList.value = []
        return
      }
      // 立即显示列表
      backupList.value = items
      // 一次性获取所有图标（一次 Go 调用，内部一次 PowerShell 批量提取）
      try {
        const icons = await GetBackupIcons()
        if (icons) {
          for (const item of items) {
            if (icons[item.name]) {
              item.icon_base64 = icons[item.name]
            }
          }
          backupList.value = [...backupList.value]
        }
      } catch {
        // 图标加载失败不影响列表
      }
    } catch (e) {
      console.error('listBackups error:', e)
      backupList.value = []
    }
  }
}

function askDelete(name) {
  confirmDeleteTarget.value = name
}

async function confirmDelete() {
  const name = confirmDeleteTarget.value
  if (!name) return
  const ok = await DeleteBackup(name)
  confirmDeleteTarget.value = ''
  if (ok) {
    showToast(t('desktopLock.deleteBackupToast').replace('{name}', name), 'success')
    backupList.value = await ListBackups()
    refreshStatus()
  } else {
    showToast(t('desktopLock.deleteBackupFail'), 'error')
  }
}

function cancelDelete() {
  confirmDeleteTarget.value = ''
}

// ── 密码验证 ──

async function verifyAndLock() {
  pwdError.value = ''
  showPwdDialog.value = true
}

async function confirmPwd() {
  pwdError.value = ''
  const ok = await VerifyPassword(pwdInput.value)
  if (ok) {
    showPwdDialog.value = false
    pwdInput.value = ''
    if (locked.value) await doUnlock()
    else await doLock()
  } else {
    pwdError.value = t('desktopLock.pwdWrong')
  }
}

function cancelPwd() {
  showPwdDialog.value = false
  pwdInput.value = ''
  pwdError.value = ''
}

onMounted(refreshStatus)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h1 class="page-title">
        <IconLock :size="24" aria-hidden="true" />
        {{ t('desktopLock.title') }}
      </h1>
      <p class="page-desc">{{ t('desktopLock.desc') }}</p>
    </div>

    <div class="status-banner" :class="{ locked }" role="status" :aria-label="t('desktopLock.title')">
      <span class="status-dot" aria-hidden="true"></span>
      <span class="status-label">{{ statusText }}</span>
    </div>

    <div class="stats-grid" style="margin-bottom: 20px;">
      <div class="stat-card">
        <span class="stat-num">{{ backupCount }}</span>
        <span class="stat-label">{{ t('desktopLock.statsBackup') }}</span>
      </div>
      <div class="stat-card">
        <span class="stat-num">{{ desktopCount }}</span>
        <span class="stat-label">{{ t('desktopLock.statsDesktop') }}</span>
      </div>
      <div v-if="missing.length" class="stat-card">
        <span class="stat-num danger">{{ missing.length }}</span>
        <span class="stat-label">{{ t('desktopLock.statsMissing') }}</span>
      </div>
    </div>

    <transition name="toast">
      <div v-if="toast" class="toast" :class="'toast-' + toastType" style="margin-bottom: 16px;" role="alert">
        {{ toast }}
      </div>
    </transition>

    <div class="card" style="margin-bottom: 16px;">
      <div style="display: flex; gap: 12px;">
        <button
          v-if="!locked"
          class="btn btn-primary" style="flex:1;"
          @click="verifyAndLock"
          :aria-label="t('desktopLock.lock')"
        >{{ t('desktopLock.lock') }}</button>
        <button
          v-if="locked"
          class="btn btn-success" style="flex:1;"
          @click="verifyAndLock"
          :aria-label="t('desktopLock.unlock')"
        >{{ t('desktopLock.unlock') }}</button>
      </div>
    </div>

    <div class="card">
      <h2 class="section-title">{{ t('desktopLock.shortcut') }}</h2>
      <div style="display: flex; gap: 8px; flex-wrap: wrap;">
        <button class="btn btn-outline btn-sm" @click="doBackup">{{ t('desktopLock.backup') }}</button>
        <button class="btn btn-outline btn-sm" @click="doRestore">{{ t('desktopLock.restore') }}</button>
        <button class="btn btn-outline btn-sm" @click="listBackups">
          {{ showBackupList ? t('common.close') : t('desktopLock.viewBackups') }}
        </button>
      </div>

      <!-- ═══ 备份列表 ═══ -->
      <div v-if="showBackupList" style="margin-top:16px;border-top:1px solid var(--border-default);padding-top:16px;">
        <p style="font-size:13px;font-weight:600;color:var(--text-secondary);margin-bottom:12px;">
          {{ t('desktopLock.backupList') }}（{{ backupList.length }}）
        </p>

        <!-- 有备份 -->
        <div v-if="backupList.length > 0" class="backup-table">
          <div v-for="item in backupList" :key="item.name" class="backup-row">
            <span v-if="!item.icon_base64" class="backup-icon">🔗</span>
            <img v-else :src="item.icon_base64" class="backup-img" alt="" />
            <div class="backup-info">
              <span class="backup-name" :title="item.name">{{ item.name }}</span>
              <span class="backup-time">{{ item.mod_time }}</span>
            </div>
            <button class="btn-delete" @click="askDelete(item.name)" :title="t('common.delete')">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                   stroke-width="2" stroke-linecap="round">
                <polyline points="3 6 5 6 21 6"/>
                <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                <path d="M10 11v6"/><path d="M14 11v6"/>
                <path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/>
              </svg>
            </button>
          </div>
        </div>

        <!-- 无备份 -->
        <div v-else style="font-size:13px;color:var(--text-placeholder);padding:20px 0;text-align:center;">
          {{ t('desktopLock.noBackups') }}
        </div>
      </div>

      <!-- 缺失列表 -->
      <div v-if="missing.length > 0" style="margin-top: 14px;">
        <p style="font-size:12px;color:var(--color-danger);margin-bottom:6px;">
          {{ t('desktopLock.missingTitle').replace('{n}', missing.length) }}
        </p>
        <div v-for="name in missing.slice(0, 10)" :key="name"
             style="font-size:12px;color:var(--text-muted);padding:3px 0;border-bottom:1px solid var(--border-default);">
          {{ name }}
        </div>
        <div v-if="missing.length > 10" style="font-size:12px;color:var(--text-placeholder);margin-top:4px;">
          {{ t('desktopLock.missingMore').replace('{n}', missing.length - 10) }}
        </div>
      </div>
    </div>

    <!-- ═══ 密码验证弹窗 ═══ -->
    <div v-if="showPwdDialog" class="overlay" role="dialog" aria-modal="true" :aria-label="locked ? t('desktopLock.pwdDialogUnlock') : t('desktopLock.pwdDialogLock')">
      <div class="dialog">
        <h3>{{ locked ? t('desktopLock.pwdDialogUnlock') : t('desktopLock.pwdDialogLock') }}</h3>
        <p>{{ t('desktopLock.pwdDialogTitle') }}</p>
        <input v-model="pwdInput" class="input" type="password" :placeholder="t('desktopLock.pwdDialogTitle')"
               aria-label="password" @keyup.enter="confirmPwd" autofocus />
        <!-- 错误提示：显示在弹窗内部 -->
        <p v-if="pwdError" class="pwd-error">{{ pwdError }}</p>
        <div style="display:flex;gap:8px;justify-content:flex-end;margin-top:16px;">
          <button class="btn btn-outline btn-sm" @click="cancelPwd">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary btn-sm" @click="confirmPwd">{{ t('common.confirm') }}</button>
        </div>
      </div>
    </div>

    <!-- ═══ 删除确认弹窗 ═══ -->
    <div v-if="confirmDeleteTarget" class="overlay" role="dialog" aria-modal="true" :aria-label="t('desktopLock.deleteConfirm')">
      <div class="dialog dialog-sm">
        <h3>{{ t('desktopLock.deleteConfirm') }}</h3>
        <p style="margin:8px 0;font-size:14px;color:var(--text-secondary);word-break:break-all;">
          {{ confirmDeleteTarget }}
        </p>
        <div style="display:flex;gap:8px;justify-content:flex-end;margin-top:20px;">
          <button class="btn btn-outline btn-sm" @click="cancelDelete">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger btn-sm" @click="confirmDelete">{{ t('desktopLock.deleteConfirmBtn') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toast-enter-active { animation: slideDown 0.25s ease; }
.toast-leave-active { animation: slideDown 0.2s ease reverse; }

/* ── 密码弹窗错误提示 ── */
.pwd-error {
  margin-top: 10px;
  font-size: 13px;
  color: var(--color-danger);
  background: rgba(229, 72, 77, 0.08);
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid rgba(229, 72, 77, 0.2);
}

/* ── 备份列表表格 ── */
.backup-table {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.backup-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 8px;
  transition: background 0.15s;
}
.backup-row:hover {
  background: var(--bg-nav-hover);
}
.backup-icon {
  font-size: 20px;
  line-height: 1;
  width: 28px;
  text-align: center;
  flex-shrink: 0;
}
.backup-img {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  object-fit: contain;
}
.backup-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.backup-name {
  font-size: 13px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.backup-time {
  font-size: 11px;
  color: var(--text-placeholder);
}
.btn-delete {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
}
.btn-delete:hover {
  background: rgba(229, 72, 77, 0.1);
  color: var(--color-danger);
}

/* ── 小弹窗 ── */
.dialog-sm {
  max-width: 380px;
}
</style>
