<script setup>
import { ref, onMounted } from 'vue'
import Sidebar from './components/Sidebar.vue'
import SettingsModal from './views/Settings.vue'
import * as theme from './theme.js'
import * as locale from './locale.js'
import { EventsOn } from '../wailsjs/runtime/runtime.js'
import { VerifyPassword } from '../wailsjs/go/desktoplock/API'
import { ConfirmQuit } from '../wailsjs/go/main/App'

const showSettings = ref(false)
const showQuitPwd = ref(false)
const quitPwdInput = ref('')
const quitPwdError = ref('')
const toast = ref('')
const toastType = ref('info')

function showToast(msg, type = 'info') {
  toast.value = msg
  toastType.value = type
  setTimeout(() => { toast.value = '' }, 4000)
}

onMounted(() => {
  locale.init()
  theme.init()

  // 监听后端退出请求
  EventsOn('request-quit', () => {
    showQuitPwd.value = true
    quitPwdInput.value = ''
    quitPwdError.value = ''
  })

  // 监听后端统一通知
  EventsOn('notify', (payload) => {
    if (payload && payload.message) {
      showToast(payload.message, payload.type || 'info')
    }
  })
})

async function confirmQuitPwd() {
  const ok = await VerifyPassword(quitPwdInput.value)
  if (ok) {
    showQuitPwd.value = false
    ConfirmQuit()
  } else {
    quitPwdError.value = '密码错误'
  }
}

function cancelQuit() {
  showQuitPwd.value = false
  quitPwdInput.value = ''
  quitPwdError.value = ''
}

function openSettings() {
  showSettings.value = true
}

function closeSettings() {
  showSettings.value = false
}
</script>

<template>
  <div class="app-layout">
    <Sidebar @show-settings="openSettings" />
    <main class="main-content">
      <router-view />
    </main>
    <SettingsModal v-if="showSettings" @close="closeSettings" />

    <!-- ═══ 统一 Toast 通知 ═══ -->
    <transition name="toast">
      <div v-if="toast" class="toast-global toast" :class="'toast-' + toastType" role="alert">
        {{ toast }}
      </div>
    </transition>

    <!-- ═══ 退出密码验证弹窗 ═══ -->
    <div v-if="showQuitPwd" class="overlay" role="dialog" aria-modal="true" aria-label="退出验证">
      <div class="dialog">
        <h3>确认退出</h3>
        <p>请输入管理密码才能退出</p>
        <input v-model="quitPwdInput" class="input" type="password"
               placeholder="请输入管理密码" @keyup.enter="confirmQuitPwd" autofocus />
        <p v-if="quitPwdError" style="margin-top:8px;font-size:13px;color:var(--color-danger);">{{ quitPwdError }}</p>
        <div style="display:flex;gap:8px;justify-content:flex-end;margin-top:16px;">
          <button class="btn btn-outline btn-sm" @click="cancelQuit">取消</button>
          <button class="btn btn-primary btn-sm" @click="confirmQuitPwd">退出</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}
.main-content {
  flex: 1;
  overflow-y: auto;
  padding: 32px;
}

</style>
