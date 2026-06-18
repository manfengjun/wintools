<script setup>
import { ref, onMounted } from 'vue'
import IconLock from '../components/icons/IconLock.vue'
import { useT } from '../locale.js'
import { Status, Lock, Unlock, Backup, Restore, VerifyPassword, ChangePassword, IsDefaultPassword } from '../../wailsjs/go/desktoplock/API'

const locked = ref(false)
const backupCount = ref(0)
const desktopCount = ref(0)
const missing = ref([])
const statusText = ref('')
const showPwdDialog = ref(false)
const showChangePwd = ref(false)
const pwdInput = ref('')
const oldPwd = ref('')
const newPwd1 = ref('')
const newPwd2 = ref('')
const toast = ref('')
const toastType = ref('info')
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

async function verifyAndLock() {
  const isDefault = await IsDefaultPassword()
  if (isDefault) {
    showToast(t('desktopLock.pwdDefaultHint'), 'info')
    showChangePwd.value = true
    return
  }
  showPwdDialog.value = true
}

async function confirmPwd() {
  const ok = await VerifyPassword(pwdInput.value)
  if (ok) {
    showPwdDialog.value = false
    pwdInput.value = ''
    if (locked.value) await doUnlock()
    else await doLock()
  } else {
    showToast(t('desktopLock.pwdWrong'), 'error')
  }
}

async function confirmChangePwd() {
  if (newPwd1.value !== newPwd2.value) {
    showToast(t('desktopLock.pwdMismatch'), 'error')
    return
  }
  const [ok, msgText] = await ChangePassword(oldPwd.value, newPwd1.value)
  if (ok) {
    showToast(t('desktopLock.pwdChanged'), 'success')
    showChangePwd.value = false
    oldPwd.value = ''; newPwd1.value = ''; newPwd2.value = ''
  } else {
    showToast(msgText, 'error')
  }
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

    <div class="card" style="margin-bottom: 16px;">
      <h2 class="section-title">{{ t('desktopLock.shortcut') }}</h2>
      <div style="display: flex; gap: 8px;">
        <button class="btn btn-outline btn-sm" @click="doBackup">{{ t('desktopLock.backup') }}</button>
        <button class="btn btn-outline btn-sm" @click="doRestore">{{ t('desktopLock.restore') }}</button>
      </div>
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

    <div class="card">
      <h2 class="section-title">{{ t('desktopLock.settings') }}</h2>
      <button class="btn btn-outline btn-sm" @click="showChangePwd = !showChangePwd">
        {{ t('desktopLock.changePwd') }}
      </button>
      <div v-if="showChangePwd" style="margin-top: 16px; display: flex; flex-direction: column; gap: 10px; max-width: 300px;">
        <label><span class="label">{{ t('desktopLock.currentPwd') }}</span>
          <input v-model="oldPwd" class="input" type="password" autocomplete="current-password" /></label>
        <label><span class="label">{{ t('desktopLock.newPwd') }}</span>
          <input v-model="newPwd1" class="input" type="password" autocomplete="new-password" /></label>
        <label><span class="label">{{ t('desktopLock.confirmPwd') }}</span>
          <input v-model="newPwd2" class="input" type="password" autocomplete="new-password" /></label>
        <div style="display:flex;gap:8px;">
          <button class="btn btn-primary btn-sm" @click="confirmChangePwd">{{ t('common.save') }}</button>
          <button class="btn btn-outline btn-sm" @click="showChangePwd = false">{{ t('common.cancel') }}</button>
        </div>
      </div>
    </div>

    <div v-if="showPwdDialog" class="overlay" @click.self="showPwdDialog = false"
         role="dialog" aria-modal="true" :aria-label="locked ? t('desktopLock.pwdDialogUnlock') : t('desktopLock.pwdDialogLock')">
      <div class="dialog">
        <h3>{{ locked ? t('desktopLock.pwdDialogUnlock') : t('desktopLock.pwdDialogLock') }}</h3>
        <p>{{ t('desktopLock.pwdDialogTitle') }}</p>
        <input v-model="pwdInput" class="input" type="password" :placeholder="t('desktopLock.pwdDialogTitle')"
               aria-label="password" @keyup.enter="confirmPwd" autofocus />
        <div style="display:flex;gap:8px;justify-content:flex-end;margin-top:16px;">
          <button class="btn btn-outline btn-sm" @click="showPwdDialog = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary btn-sm" @click="confirmPwd">{{ t('common.confirm') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toast-enter-active { animation: slideDown 0.25s ease; }
.toast-leave-active { animation: slideDown 0.2s ease reverse; }
</style>
