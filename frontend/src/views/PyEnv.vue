<script setup>
import { ref, reactive, onMounted, onUnmounted, nextTick, computed } from 'vue'
import IconPython from '../components/icons/IconPython.vue'
import { useT } from '../locale.js'
import { initialProgress, applyProgress } from './pyEnvProgress.js'

// 使用 Wails 生成的绑定模块
import { AvailablePackages, CheckStatus, InstallWithElevation } from '../../wailsjs/go/pyenv/InstallerAPI'

const t = useT()

const installed = ref(false)
const pythonExe = ref('')
const version = ref('')
const pipInstalled = ref(false)
const installing = ref(false)
const log = ref([])
const logContainer = ref(null)
const packages = ref([])

// Progress reducer state
const progress = reactive(initialProgress())

let eventsCancel = null

// ── 包列表 ──
async function loadPackages() {
  try {
    const pkgs = await AvailablePackages()
    packages.value = pkgs.map(p => ({ ...p, checked: p.default_on }))
  } catch (e) {
    addLog('加载包列表失败: ' + String(e), 'error')
  }
}

// ── 日志（用于 reducer 未覆盖的 UI 日志）──
function addLog(msg, type = 'info') {
  log.value.push({ msg, type, time: new Date().toLocaleTimeString() })
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
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
    addLog(t('pyEnv.logStatusFailed') + ': ' + String(e), 'error')
  }
}

function listenProgress() {
  if (typeof window.runtime !== 'undefined' && window.runtime.EventsOn) {
    window.runtime.EventsOn('pyenv:progress', (data) => {
      // Apply event through the reducer
      const event = {
        step: data.step || '',
        message: data.message || '',
        percent: data.percent,
        done: data.done,
        error: data.error,
        package: data.package,
        package_status: data.package_status,
      }
      const next = applyProgress(progress, event)
      Object.assign(progress, next)

      installing.value = next.installing

      if (data.message) addLog(data.message, data.error ? 'error' : data.done ? 'success' : 'info')
      if (data.error) addLog('❌ ' + data.error, 'error')
      if (data.done) { installing.value = false; checkStatus() }
    })
    eventsCancel = () => window.runtime.EventsOff('pyenv:progress')
  }
}

async function startInstall() {
  installing.value = true
  installing.value = true
  log.value = []

  // Reset progress via reducer
  const selected = packages.value.filter(p => p.checked).map(p => p.id)
  const fresh = initialProgress(selected)
  Object.assign(progress, fresh)

  addLog(t('pyEnv.logStart'))
  addLog('选择包: ' + (selected.length > 0 ? selected.join(', ') : '（无）'))

  try {
    const result = await InstallWithElevation(selected)
    if (result && result.error) addLog('❌ ' + result.error, 'error')
    else if (result && result.done) addLog('✅ ' + (result.message || t('pyEnv.logDone')), 'success')
  } catch (e) {
    addLog('❌ ' + t('pyEnv.logFailed') + ': ' + String(e), 'error')
  }
  installing.value = false
  await checkStatus()
}

function toggleAll(checked) {
  packages.value.forEach(p => { p.checked = checked })
}

// Show the progress card whenever we have active or visible events
const showProgress = computed(() => progress.visible || progress.logs.length > 0)

onMounted(() => { checkStatus(); listenProgress(); loadPackages() })
onUnmounted(() => { if (eventsCancel) eventsCancel() })
</script>

<template>
  <div class="page" style="max-width:600px;">
    <div class="page-header">
      <h1 class="page-title">
        <IconPython :size="24" aria-hidden="true" /> {{ t('pyEnv.title') }}
      </h1>
      <p class="page-desc">{{ t('pyEnv.desc') }}</p>
    </div>

    <!-- Status -->
    <div class="card" style="margin-bottom:20px;">
      <div style="display:flex;align-items:center;gap:14px;margin-bottom:12px;">
        <div class="status-dot-lg" :class="{ ok: installed, none: !installed }" aria-hidden="true"></div>
        <div>
          <div style="font-weight:600;">{{ installed ? t('pyEnv.installed') : t('pyEnv.notInstalled') }}</div>
          <div v-if="version" style="font-size:13px;color:var(--text-muted);">{{ version.trim() }}</div>
        </div>
      </div>
      <div v-if="installed" style="font-size:13px;color:var(--text-secondary);">
        <span style="color:var(--text-placeholder);">{{ t('pyEnv.path') }}：</span><code>{{ pythonExe }}</code>
      </div>
    </div>

    <!-- Progress + Log (visible while installing OR after completion/error) -->
    <div v-if="showProgress" class="card" style="margin-bottom:20px;">
      <div style="margin-bottom:12px;">
        <div style="font-weight:600;font-size:14px;margin-bottom:4px;">
          {{ progress.message || (installing ? t('common.loading') : '') }}
        </div>
        <!-- Determinate progress bar -->
        <div v-if="progress.step !== 'install-python'" class="progress-bar">
          <div class="progress-fill" :style="{width: progress.percent + '%'}" role="progressbar"
               :aria-valuenow="progress.percent" aria-valuemin="0" aria-valuemax="100"></div>
        </div>
        <!-- Indeterminate progress bar during install-python -->
        <div v-else class="progress-bar indeterminate">
          <div class="progress-fill" role="progressbar" aria-label="安装进行中..."></div>
        </div>
      </div>

      <!-- Package statuses -->
      <div v-if="Object.keys(progress.packages).length > 0" style="margin-bottom:12px;">
        <div class="pkg-status-grid">
          <div v-for="(status, name) in progress.packages" :key="name" class="pkg-status-item" :class="'pkg-' + status">
            <span class="pkg-status-icon">{{ status === 'success' ? '✓' : status === 'failed' ? '✗' : status === 'installing' ? '⟳' : '○' }}</span>
            <span class="pkg-status-name">{{ name }}</span>
          </div>
        </div>
      </div>

      <!-- Log box -->
      <div class="log-box" ref="logContainer">
        <div v-for="(entry,i) in log" :key="i" class="log-line" :class="entry.type">
          <span class="log-time">{{ entry.time }}</span><span>{{ entry.msg }}</span>
        </div>
      </div>
    </div>

    <!-- Package Selection -->
    <div class="card" style="margin-bottom:20px;">
      <h2 class="section-title" style="display:flex;align-items:center;justify-content:space-between;">
        <span>安装包选择</span>
        <span style="font-size:12px;color:var(--text-muted);font-weight:400;">
          <label style="cursor:pointer;margin-right:12px;">
            <input type="checkbox" :checked="packages.length > 0 && packages.every(p=>p.checked)"
                   @change="e => toggleAll(e.target.checked)" /> 全选
          </label>
          {{ packages.filter(p=>p.checked).length }}/{{ packages.length }}
        </span>
      </h2>
      <div class="pkg-grid">
        <label v-for="pkg in packages" :key="pkg.id" class="pkg-item" :class="{ disabled: installing }">
          <input type="checkbox" v-model="pkg.checked" :disabled="installing" class="pkg-check" />
          <span class="pkg-checkmark" aria-hidden="true"></span>
          <span class="pkg-info">
            <span class="pkg-name">{{ pkg.name }}</span>
            <span class="pkg-desc">{{ pkg.description }}</span>
          </span>
        </label>
      </div>
    </div>

    <!-- Install Button -->
    <div class="card">
      <p style="font-size:13px;color:var(--text-muted);margin-bottom:16px;line-height:1.7;">
        {{ t('pyEnv.installingDesc').replace('{path}', 'Python 官方安装目录（所有用户）') }}
      </p>
      <button class="btn" :class="installed?'btn-outline':'btn-primary'" :disabled="installing"
              @click="startInstall" :aria-label="installing ? t('pyEnv.installing') : (installed ? t('pyEnv.reinstallBtn') : t('pyEnv.installBtn'))">
        <template v-if="installing"><span class="spinner" aria-hidden="true"></span>{{ t('pyEnv.installing') }}</template>
        <template v-else>{{ installed ? t('pyEnv.reinstallBtn') : t('pyEnv.installBtn') }}</template>
      </button>
    </div>
  </div>
</template>

<style scoped>
.status-dot-lg {
  width:16px;height:16px;border-radius:50%;flex-shrink:0;
  transition:background var(--transition-fast);
}
.status-dot-lg.ok { background: var(--color-success); }
.status-dot-lg.none { background: var(--text-placeholder); }

.log-box {
  max-height:220px;overflow-y:auto;
  background:var(--bg-code);border-radius:var(--radius-sm);padding:12px;
  font-family:var(--font-mono);font-size:12px;line-height:1.7;
}
.log-line { color:var(--text-secondary); }
.log-line.success { color:var(--color-success); }
.log-line.error   { color:var(--color-danger);  }
.log-line.info    { color:var(--text-secondary); }
.log-time { color:var(--text-placeholder);margin-right:8px;user-select:none; }

.spinner {
  display:inline-block;width:14px;height:14px;
  border:2px solid rgba(255,255,255,0.3);border-top-color:#fff;
  border-radius:50%;animation:spin 0.6s linear infinite;
}
@keyframes spin { to { transform:rotate(360deg); } }

/* ── 包选择网格 ── */
.pkg-grid {
  display:grid;
  grid-template-columns:repeat(auto-fill, minmax(220px, 1fr));
  gap:8px;
}
.pkg-item {
  display:flex;
  align-items:center;
  gap:10px;
  padding:10px 12px;
  border:1px solid var(--border-default);
  border-radius:var(--radius-sm);
  cursor:pointer;
  transition:all var(--transition-fast);
  user-select:none;
}
.pkg-item:hover { border-color:var(--color-primary); }
.pkg-item.disabled { opacity:0.6; cursor:not-allowed; }
.pkg-item:has(.pkg-check:checked) {
  border-color:var(--color-primary);
  background:var(--accent-bg);
}
.pkg-check {
  position:absolute;opacity:0;width:0;height:0;
}
.pkg-checkmark {
  width:18px;height:18px;border-radius:4px;
  border:2px solid var(--border-input);
  display:flex;align-items:center;justify-content:center;
  flex-shrink:0;transition:all var(--transition-fast);
}
.pkg-checkmark::after {
  content:'✓';
  font-size:12px;font-weight:700;
  color:#fff;transform:scale(0);
  transition:transform var(--transition-fast);
}
.pkg-check:checked + .pkg-checkmark {
  background:var(--color-primary);border-color:var(--color-primary);
}
.pkg-check:checked + .pkg-checkmark::after { transform:scale(1); }
.pkg-check:focus-visible + .pkg-checkmark {
  box-shadow:0 0 0 3px rgba(79,110,247,0.2);
}
.pkg-info {
  display:flex;flex-direction:column;gap:1px;
  min-width:0;
}
.pkg-name {
  font-size:13px;font-weight:600;color:var(--text-primary);
}
.pkg-desc {
  font-size:11px;color:var(--text-muted);
  overflow:hidden;text-overflow:ellipsis;white-space:nowrap;
}

/* ── 进度包状态网格 ── */
.pkg-status-grid {
  display:flex;
  flex-wrap:wrap;
  gap:6px;
}
.pkg-status-item {
  display:inline-flex;
  align-items:center;
  gap:4px;
  padding:3px 8px;
  border-radius:var(--radius-sm);
  font-size:12px;
  background:var(--bg-code);
}
.pkg-status-icon {
  font-size:12px;
  width:14px;
  text-align:center;
}
.pkg-pending   { color:var(--text-placeholder); }
.pkg-installing { color:var(--color-primary); }
.pkg-success   { color:var(--color-success); }
.pkg-failed    { color:var(--color-danger); }

/* ── 不确定进度条 ── */
.progress-bar.indeterminate .progress-fill {
  width:30%;
  animation: indeterminate 1.5s ease-in-out infinite;
}
@keyframes indeterminate {
  0%   { transform:translateX(-100%); }
  100% { transform:translateX(400%); }
}
</style>
