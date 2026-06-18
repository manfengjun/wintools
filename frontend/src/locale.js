/**
 * locale.js — 国际化和本地化引擎
 *
 * 支持：中文 / 英文，自动检测系统语言，持久化偏好。
 * 使用 useT() 返回响应式 t() 函数，切换语言时自动重新渲染。
 */

import { ref } from 'vue'
import zh from './locales/zh.js'
import en from './locales/en.js'

const messages = { zh, en }
const STORAGE_KEY = 'codepower-studio-locale'

// ── 响应式版本戳（切换语言时自增，驱动 Vue 重新渲染） ──
const _tick = ref(0)

// ── 当前语言 ──
let _currentLocale = 'zh'

// ── 检测系统语言 ──
function detectLanguage() {
  const lang = (navigator.language || navigator.userLanguage || 'zh').toLowerCase()
  return lang.startsWith('zh') ? 'zh' : 'en'
}

// ── 读写偏好 ──
function loadPref() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw === 'zh' || raw === 'en') return raw
  } catch { /* ignore */ }
  return 'auto'
}

function savePref(locale) {
  localStorage.setItem(STORAGE_KEY, locale)
}

// ── 解析嵌套 key ──
function resolve(obj, key) {
  return key.split('.').reduce((acc, part) => acc?.[part], obj)
}

// ── 扁平化的 t 函数（供 useT 返回） ──
function translate(key, fallback) {
  const msg = resolve(messages[_currentLocale], key)
  if (msg !== undefined) return msg
  const enMsg = resolve(messages.en, key)
  if (enMsg !== undefined) return enMsg
  return fallback ?? key
}

// ── 响应式翻译钩子 ──
// 在组件中调用 useT() 返回 t 函数，模板中 {{ t('key') }} 会在语言切换时自动更新。
export function useT() {
  // 引用 _tick 使其成为响应式依赖
  const tick = _tick
  return function t(key, fallback) {
    // 读取 tick.value 建立响应式依赖
    void tick.value
    return translate(key, fallback)
  }
}

// ── 初始化 ──
export function init() {
  const pref = loadPref()
  _currentLocale = pref === 'auto' ? detectLanguage() : pref
  document.documentElement.setAttribute('lang', _currentLocale)
}

// ── 切换语言 ──
export function setLocale(locale) {
  _currentLocale = locale
  savePref(locale)
  document.documentElement.setAttribute('lang', _currentLocale)
  _tick.value++ // 触发所有 useT() 消费者重新渲染
}

export function getLocale() { return _currentLocale }
export { detectLanguage, loadPref as loadLocalePref }
