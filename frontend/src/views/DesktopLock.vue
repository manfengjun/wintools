<script setup>
import { ref, onMounted } from 'vue'
import IconLock from '../components/icons/IconLock.vue'
import { useT } from '../locale.js'
import { Status, Lock, Unlock, Backup, Restore, VerifyPassword, ListBackups, DeleteBackup } from '../../wailsjs/go/desktoplock/API'
import { EventsEmit } from '../../wailsjs/runtime/runtime.js'
import { normalizeBackupItems } from './desktopLockBackups.js'

const locked = ref(false)
const backupCount = ref(0)
const desktopCount = ref(0)
const missing = ref([])
const statusText = ref('')
const showPwdDialog = ref(false)
const pwdInput = ref('')
const pwdError = ref('')
const backupList = ref([])
const showBackupList = ref(false)
const confirmDeleteTarget = ref('')
const t = useT()

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
  EventsEmit('notify', { message: t('desktopLock.lockedToast'), type: 'success' })
  refreshStatus()
}

async function doUnlock() {
  await Unlock()
  EventsEmit('notify', { message: t('desktopLock.unlockedToast'), type: 'success' })
  refreshStatus()
}

async function doBackup() {
  const r = await Backup()
  EventsEmit('notify', { message: t('desktopLock.backupToast').replace('{n}', r.ok), type: 'success' })
  refreshStatus()
}

async function doRestore() {
  const r = await Restore()
  EventsEmit('notify', { message: t('desktopLock.restoreToast').replace('{n}', r.restored), type: 'success' })
  refreshStatus()
}

async function listBackups() {
  showBackupList.value = !showBackupList.value
  if (!showBackupList.value) {
    return
  }

  try {
    const items = await ListBackups()
    if (!items || !Array.isArray(items)) {
      backupList.value = []
      return
    }

    backupList.value = normalizeBackupItems(items)
  } catch (e) {
    console.error('listBackups error:', e)
    backupList.value = []
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
    EventsEmit('notify', { message: t('desktopLock.deleteBackupToast').replace('{name}', name), type: 'success' })
    backupList.value = normalizeBackupItems(await ListBackups())
    refreshStatus()
  } else {
    EventsEmit('notify', { message: t('desktopLock.deleteBackupFail'), type: 'error' })
  }
}

function cancelDelete() {
  confirmDeleteTarget.value = ''
}

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

    <div class="stats-grid mb-20">
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

    <!-- Toast 已迁移至 App.vue 全局 -->

    <div class="card mb-16">
      <div class="flex-row gap-12">
        <button
          v-if="!locked"
          class="btn btn-primary btn-flex"
          data-demo-id="lock-toggle"
          @click="verifyAndLock"
          :aria-label="t('desktopLock.lock')"
        >{{ t('desktopLock.lock') }}</button>
        <button
          v-if="locked"
          class="btn btn-success btn-flex"
          data-demo-id="lock-toggle"
          @click="verifyAndLock"
          :aria-label="t('desktopLock.unlock')"
        >{{ t('desktopLock.unlock') }}</button>
      </div>
    </div>

    <div class="card">
      <h2 class="section-title">{{ t('desktopLock.shortcut') }}</h2>
      <div class="flex-wrap">
        <button class="btn btn-outline btn-sm" @click="doBackup">{{ t('desktopLock.backup') }}</button>
        <button class="btn btn-outline btn-sm" @click="doRestore">{{ t('desktopLock.restore') }}</button>
        <button class="btn btn-outline btn-sm" @click="listBackups">
          {{ showBackupList ? t('common.close') : t('desktopLock.viewBackups') }}
        </button>
      </div>

      <transition name="backup-panel">
        <div v-if="showBackupList" class="mt-16">
          <hr class="divider" />
          <p class="backup-list-title">
          {{ t('desktopLock.backupList') }}（{{ backupList.length }}）
        </p>

        <div v-if="backupList.length > 0" class="backup-table">
          <div v-for="item in backupList" :key="item.name" class="backup-row">
            <span v-if="!item.icon_base64" class="backup-icon" aria-hidden="true">
              <svg viewBox="0 0 28 28" fill="none">
                <rect x="5" y="8" width="18" height="15" rx="5" />
                <path d="M14 8V5" />
                <circle cx="10.5" cy="15" r="1.3" />
                <circle cx="17.5" cy="15" r="1.3" />
                <path d="M11 19h6" />
              </svg>
            </span>
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

        <div v-else class="empty-state">
          {{ t('desktopLock.noBackups') }}
        </div>
        </div>
      </transition>

      <div v-if="missing.length > 0" class="mt-14">
        <p class="missing-title">
          {{ t('desktopLock.missingTitle').replace('{n}', missing.length) }}
        </p>
        <div v-for="name in missing.slice(0, 10)" :key="name"
             class="missing-item">
          {{ name }}
        </div>
        <div v-if="missing.length > 10" class="missing-more">
          {{ t('desktopLock.missingMore').replace('{n}', missing.length - 10) }}
        </div>
      </div>
    </div>

    <div v-if="showPwdDialog" class="overlay" role="dialog" aria-modal="true" :aria-label="locked ? t('desktopLock.pwdDialogUnlock') : t('desktopLock.pwdDialogLock')">
      <div class="dialog">
        <h3>{{ locked ? t('desktopLock.pwdDialogUnlock') : t('desktopLock.pwdDialogLock') }}</h3>
        <p>{{ t('desktopLock.pwdDialogTitle') }}</p>
        <input v-model="pwdInput" class="input" type="password" data-demo-id="password-input" :placeholder="t('desktopLock.pwdDialogTitle')"
               aria-label="password" @keyup.enter="confirmPwd" autofocus />
        <p v-if="pwdError" class="pwd-error">{{ pwdError }}</p>
        <div style="display:flex;gap:8px;justify-content:flex-end;margin-top:16px;">
          <button class="btn btn-outline btn-sm" @click="cancelPwd">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary btn-sm" data-demo-id="password-confirm" @click="confirmPwd">{{ t('common.confirm') }}</button>
        </div>
      </div>
    </div>

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
.pwd-error {
  margin-top: 10px;
  font-size: 13px;
  color: var(--color-danger);
  background: rgba(229, 72, 77, 0.08);
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid rgba(229, 72, 77, 0.2);
}

/* 备份列表展开/收起动画 */
.backup-panel-enter-active,
.backup-panel-leave-active {
  transition: max-height 0.25s ease, opacity 0.25s ease;
  overflow: hidden;
}
.backup-panel-enter-from,
.backup-panel-leave-to {
  max-height: 0;
  opacity: 0;
}
.backup-panel-enter-to,
.backup-panel-leave-from {
  max-height: 600px;
  opacity: 1;
}

/* 密码弹窗进入动画 */
.dialog-enter-active {
  animation: dialogIn 0.2s ease;
}
@keyframes dialogIn {
  from { opacity: 0; transform: scale(0.95) translateY(4px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}

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
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
  flex-shrink: 0;
  color: #e14aa0;
  background: linear-gradient(135deg, rgba(255, 119, 190, 0.18), rgba(98, 211, 255, 0.14));
  box-shadow: inset 0 0 0 1px rgba(225, 74, 160, 0.18);
}
.backup-icon svg {
  width: 22px;
  height: 22px;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.backup-icon rect,
.backup-icon circle {
  stroke: currentColor;
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

.dialog-sm {
  max-width: 380px;
}

/* ── 缺失图标列表样式 ── */
.mt-14 { margin-top: 14px; }
.missing-title {
  font-size: 12px;
  color: var(--color-danger);
  margin-bottom: 6px;
}
.missing-item {
  font-size: 12px;
  color: var(--text-muted);
  padding: 3px 0;
  border-bottom: 1px solid var(--border-default);
}
.missing-more {
  font-size: 12px;
  color: var(--text-placeholder);
  margin-top: 4px;
}

</style>
