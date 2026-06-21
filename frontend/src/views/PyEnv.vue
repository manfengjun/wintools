<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import IconPython from '../components/icons/IconPython.vue'
import { useT } from '../locale.js'

import { AvailablePackages, CheckStatus, InstallPython, InstallPackages } from '../../wailsjs/go/pyenv/InstallerAPI'

const installed = ref(false)
const pythonExe = ref('')
const version = ref('')
const pipInstalled = ref(false)

const installingPython = ref(false)
const pyLog = ref([])
const pyLogContainer = ref(null)

const packages = ref([])
const installingPkgs = ref(false)
const pkgLog = ref([])
const pkgLogContainer = ref(null)
const t = useT()

let eventsCancel = null

function addLog(logArr, container, msg, type = 'info') {
  logArr.value.push({ msg, type, time: new Date().toLocaleTimeString() })
  nextTick(() => {
    if (container.value) container.value.scrollTop = container.value.scrollHeight
  })
}

async function checkStatus() {
  try {
    const s = await CheckStatus()
    installed.value = s.installed
    pythonExe.value = s.python_exe
    version.value = s.version
    pipInstalled.value = s.pip_installed
  } catch (e) {
    addLog(pyLog, pyLogContainer, t('pyEnv.logStatusFailed') + ': ' + String(e), 'error')
  }
}

async function loadPackages() {
  try {
    const pkgs = await AvailablePackages()
    packages.value = pkgs.map(p => ({ ...p, checked: p.default_on }))
  } catch (e) {
    addLog(pkgLog, pkgLogContainer, t('pyEnv.logFailed') + ': ' + String(e), 'error')
  }
}

function listenProgress() {
  if (typeof window.runtime !== 'undefined' && window.runtime.EventsOn) {
    window.runtime.EventsOn('pyenv:progress', (data) => {
      const msg = data.message || data.step || ''
      if (!msg) return
      const isErr = !!data.error
      const isDone = !!data.done
      if (data.step === 'install-package') {
        addLog(pkgLog, pkgLogContainer, (isErr ? '❌ ' : isDone ? '✅ ' : '') + msg, isErr ? 'error' : isDone ? 'success' : 'info')
      } else {
        addLog(pyLog, pyLogContainer, (isErr ? '❌ ' : isDone ? '✅ ' : '') + msg, isErr ? 'error' : 'info')
      }
    })
    eventsCancel = () => window.runtime.EventsOff('pyenv:progress')
  }
}

async function doInstallPython() {
  installingPython.value = true
  pyLog.value = []
  addLog(pyLog, pyLogContainer, t('pyEnv.startInstall'))
  try {
    const result = await InstallPython()
    if (result && result.error) addLog(pyLog, pyLogContainer, '❌ ' + result.error, 'error')
  } catch (e) {
    addLog(pyLog, pyLogContainer, '❌ ' + t('pyEnv.logFailed') + ': ' + String(e), 'error')
  }
  installingPython.value = false
  await checkStatus()
}

async function doInstallPackages() {
  const selected = packages.value.filter(p => p.checked).map(p => p.id)
  if (selected.length === 0) {
    addLog(pkgLog, pkgLogContainer, t('pyEnv.selectPackageHint'), 'error')
    return
  }
  installingPkgs.value = true
  pkgLog.value = []
  addLog(pkgLog, pkgLogContainer, t('pyEnv.startInstallPkgs').replace('{n}', selected.length))
  addLog(pkgLog, pkgLogContainer, t('pyEnv.selectedPackages') + selected.join(', '))
  try {
    const result = await InstallPackages(selected)
    if (result && result.error) addLog(pkgLog, pkgLogContainer, '❌ ' + result.error, 'error')
  } catch (e) {
    addLog(pkgLog, pkgLogContainer, '❌ ' + t('pyEnv.logFailed') + ': ' + String(e), 'error')
  }
  installingPkgs.value = false
  await checkStatus()
}

function toggleAll(checked) {
  packages.value.forEach(p => { p.checked = checked })
}

onMounted(() => { checkStatus(); listenProgress(); loadPackages() })
onUnmounted(() => { if (eventsCancel) eventsCancel() })
</script>

<template>
  <div class="page" style="max-width:680px;">

    <!-- ═══════════════════════════════════════ -->
    <!--  步骤 1：安装 Python 环境               -->
    <!-- ═══════════════════════════════════════ -->
    <div class="card card-hoverable">
      <div class="section-header">
        <IconPython :size="22" aria-hidden="true" />
        <h2 class="section-title" style="margin:0;">{{ t('pyEnv.sectionTitle') }}</h2>
        <span class="badge" :class="installed ? 'badge-ok' : 'badge-none'">
          <span class="badge-dot"></span>
          {{ installed ? t('pyEnv.statusInstalled') : t('pyEnv.statusNotInstalled') }}
        </span>
      </div>

      <p class="page-desc">{{ t('pyEnv.desc') }}</p>

      <!-- 状态详情 -->
      <div v-if="installed" class="status-grid">
        <div class="status-item">
          <span class="status-label">{{ t('pyEnv.labelVersion') }}</span>
          <code class="status-value">{{ version.trim() }}</code>
        </div>
        <div class="status-item">
          <span class="status-label">{{ t('pyEnv.labelPath') }}</span>
          <code class="status-value status-path">{{ pythonExe }}</code>
        </div>
        <div class="status-item">
          <span class="status-label">{{ t('pyEnv.labelPip') }}</span>
          <code class="status-value">{{ pipInstalled ? t('pyEnv.statusInstalled') : t('pyEnv.statusNotInstalled') }}</code>
        </div>
      </div>

      <!-- 安装日志 -->
      <div v-if="pyLog.length > 0" class="log-box" ref="pyLogContainer">
        <div v-for="(entry,i) in pyLog" :key="i" class="log-entry" :class="'log-'+entry.type">
          <span class="log-bullet"></span>
          <span class="log-time">{{ entry.time }}</span>
          <span class="log-msg">{{ entry.msg }}</span>
        </div>
      </div>

      <!-- 按钮 -->
      <button class="btn" :class="installed ? 'btn-outline' : 'btn-primary'"
              :disabled="installingPython" @click="doInstallPython">
        <span v-if="installingPython" class="spinner"></span>
        {{ installingPython ? t('pyEnv.installing') : (installed ? t('pyEnv.reinstall') : t('pyEnv.install')) }}
      </button>
    </div>

    <!-- ═══════════════════════════════════════ -->
    <!--  步骤 2：安装 Python 库                -->
    <!-- ═══════════════════════════════════════ -->
    <div class="card card-hoverable">
      <div class="section-header">
        <span class="icon-lib" aria-hidden="true">📚</span>
        <h2 class="section-title" style="margin:0;">{{ t('pyEnv.packageSection') }}</h2>
        <span v-if="!installed" class="badge badge-warn">
          <span class="badge-dot"></span>{{ t('pyEnv.needInstallFirst') }}
        </span>
        <span v-else class="badge badge-count">{{ packages.filter(p=>p.checked).length }}/{{ packages.length }}</span>
      </div>

      <p class="page-desc">{{ t('pyEnv.installingDesc') }}</p>

      <!-- 包选择 -->
      <div class="pkg-grid">
        <label v-for="pkg in packages" :key="pkg.id" class="pkg-item"
               :class="{ disabled: !installed || installingPkgs }">
          <input type="checkbox" v-model="pkg.checked"
                 :disabled="!installed || installingPkgs" class="pkg-check" />
          <span class="pkg-checkmark" aria-hidden="true"></span>
          <span class="pkg-info">
            <span class="pkg-name">{{ pkg.name }}</span>
            <span class="pkg-desc">{{ pkg.description }}</span>
          </span>
        </label>
      </div>

      <!-- 全选 -->
      <div class="select-all-row">
        <label class="select-all-label">
          <input type="checkbox"
                 :checked="packages.length > 0 && packages.every(p=>p.checked)"
                 @change="e => toggleAll(e.target.checked)"
                 :disabled="!installed || installingPkgs" />
          {{ t('pyEnv.selectAll') }}
        </label>
      </div>

      <!-- 安装日志 -->
      <div v-if="pkgLog.length > 0" class="log-box" ref="pkgLogContainer">
        <div v-for="(entry,i) in pkgLog" :key="i" class="log-entry" :class="'log-'+entry.type">
          <span class="log-bullet"></span>
          <span class="log-time">{{ entry.time }}</span>
          <span class="log-msg">{{ entry.msg }}</span>
        </div>
      </div>

      <!-- 按钮 -->
      <button class="btn btn-primary" :disabled="!installed || installingPkgs"
              @click="doInstallPackages">
        <span v-if="installingPkgs" class="spinner"></span>
        {{ installingPkgs ? t('pyEnv.installing') : t('pyEnv.installSelected') }}
      </button>
    </div>

  </div>
</template>

<style scoped>
/* ── 卡片悬浮效果 ── */
.card-hoverable {
  transition: box-shadow 0.2s, transform 0.2s;
}
.card-hoverable:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-1px);
}

/* ── 分区头部 ── */
.section-header {
  display:flex;
  align-items:center;
  gap:10px;
  margin-bottom:10px;
}

/* ── 徽标 ── */
.badge {
  display:inline-flex;
  align-items:center;
  gap:5px;
  margin-left:auto;
  font-size:12px;
  font-weight:600;
  padding:3px 10px;
  border-radius:20px;
  white-space:nowrap;
}
.badge-dot {
  width:8px;height:8px;
  border-radius:50%;
  display:inline-block;
}
.badge-ok { background:var(--color-success-bg,#e6f7e6);color:var(--color-success,#389e0d); }
.badge-ok .badge-dot { background:var(--color-success); }
.badge-none { background:var(--bg-code);color:var(--text-placeholder); }
.badge-none .badge-dot { background:var(--text-placeholder); }
.badge-warn { background:var(--color-warning-bg,#fff7e6);color:var(--color-warning,#d48806); }
.badge-warn .badge-dot { background:var(--color-warning); }
.badge-count { background:var(--accent-bg);color:var(--color-primary); }

/* ── 状态网格 ── */
.status-grid {
  display:flex;
  flex-wrap:wrap;
  gap:4px 20px;
  margin-bottom:14px;
  padding:10px 12px;
  background:var(--bg-code);
  border-radius:var(--radius-sm);
}
.status-item {
  display:flex;
  align-items:center;
  gap:6px;
  font-size:13px;
}
.status-label {
  color:var(--text-placeholder);
  font-size:12px;
}
.status-value {
  color:var(--text-secondary);
  font-size:13px;
  font-weight:500;
}
.status-path {
  max-width:280px;
  overflow:hidden;
  text-overflow:ellipsis;
  white-space:nowrap;
}

/* ── 日志时间线 ── */
.log-box {
  max-height:200px;
  overflow-y:auto;
  margin-bottom:14px;
}
.log-entry {
  display:flex;
  align-items:flex-start;
  gap:8px;
  padding:3px 0;
  font-family:var(--font-mono);
  font-size:12px;
  line-height:1.7;
  border-left:2px solid transparent;
  padding-left:8px;
  transition:border-color 0.15s;
}
.log-entry:hover {
  border-left-color:var(--border-default);
}
.log-bullet {
  flex-shrink:0;
  width:6px;height:6px;
  border-radius:50%;
  margin-top:7px;
  background:var(--text-placeholder);
}
.log-info .log-bullet { background:var(--color-primary); }
.log-success .log-bullet { background:var(--color-success); }
.log-error .log-bullet { background:var(--color-danger); }
.log-time {
  flex-shrink:0;
  color:var(--text-placeholder);
  font-size:11px;
  min-width:52px;
  user-select:none;
}
.log-msg {
  color:var(--text-secondary);
  word-break:break-all;
}
.log-success .log-msg { color:var(--color-success); }
.log-error .log-msg { color:var(--color-danger); }

/* ── 包选择网格 ── */
.pkg-grid {
  display:grid;
  grid-template-columns:repeat(auto-fill, minmax(185px, 1fr));
  gap:8px;
  margin-bottom:10px;
}
.pkg-item {
  display:flex;align-items:center;gap:10px;
  padding:9px 12px;
  border:1px solid var(--border-default);
  border-radius:var(--radius-sm);
  cursor:pointer;
  transition:all 0.15s, border-color 0.15s;
  user-select:none;
}
.pkg-item:hover:not(.disabled) {
  border-color:var(--color-primary);
  box-shadow:0 1px 4px rgba(from var(--color-primary) r g b / 0.08);
}
.pkg-item.disabled { opacity:0.5; cursor:not-allowed; }
.pkg-item:has(.pkg-check:checked) {
  border-color:var(--color-primary);
  background:var(--accent-bg);
}
.pkg-check { position:absolute;opacity:0;width:0;height:0; }
.pkg-checkmark {
  width:18px;height:18px;border-radius:4px;
  border:2px solid var(--border-input);
  display:flex;align-items:center;justify-content:center;
  flex-shrink:0;transition:all 0.15s;
}
.pkg-checkmark::after {
  content:'✓';font-size:12px;font-weight:700;
  color:#fff;transform:scale(0);
  transition:transform 0.15s;
}
.pkg-check:checked + .pkg-checkmark {
  background:var(--color-primary);border-color:var(--color-primary);
}
.pkg-check:checked + .pkg-checkmark::after { transform:scale(1); }
.pkg-check:focus-visible + .pkg-checkmark {
  box-shadow:0 0 0 3px rgba(from var(--color-primary) r g b / 0.2);
}
.pkg-info { display:flex;flex-direction:column;gap:2px;min-width:0; }
.pkg-name { font-size:13px;font-weight:600;color:var(--text-primary); }
.pkg-desc { font-size:11px;color:var(--text-muted);overflow:hidden;text-overflow:ellipsis;white-space:nowrap; }

/* ── 全选行 ── */
.select-all-row {
  display:flex;
  align-items:center;
  margin-bottom:12px;
}
.select-all-label {
  display:flex;
  align-items:center;
  gap:6px;
  font-size:13px;
  color:var(--text-muted);
  cursor:pointer;
  user-select:none;
}

.icon-lib { font-size:22px; }

/* ── 日志条目逐项进入 ── */
.log-entry {
  animation: fadeIn 0.2s ease both;
}
</style>
