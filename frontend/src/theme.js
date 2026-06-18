/**
 * theme.js — 主题与视觉风格管理系统
 *
 * 管理：浅色/深色/自动主题、视觉风格色板、字号、字体。
 * 持久化到 localStorage，启动时自动恢复。
 */

const STORAGE_KEY = 'codepower-studio-theme'

// ── 视觉风格色板 ──
const palettes = {
  graphite: { accent: '#e8590c', accentBg: '#fff4e6', name: { zh: '石墨', en: 'Graphite' } },
  aurora:   { accent: '#7c3aed', accentBg: '#f0eeff', name: { zh: '柔雾极光', en: 'Aurora' } },
  slate:    { accent: '#2563eb', accentBg: '#eef2ff', name: { zh: '精炼', en: 'Slate' } },
  carbon:   { accent: '#0d9488', accentBg: '#edfcf5', name: { zh: '深邃', en: 'Carbon' } },
}

// ── 默认设置 ──
const defaults = {
  theme: 'auto',        // 'light' | 'dark' | 'auto'
  visualStyle: 'aurora',
  fontSize: 'default',
  fontFamily: 'system',
}

// ── 字号映射 ──
const fontSizes = {
  small:   '13px',
  default: '14px',
  large:   '15px',
  xlarge:  '17px',
}

// ── 字体映射 ──
const fontFamilies = {
  system: "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif",
  msyh:   "'Microsoft YaHei','微软雅黑',-apple-system,sans-serif",
  source: "'Source Han Sans SC','思源黑体',-apple-system,sans-serif",
}

// ── 媒体查询 ──
let darkModeMq = null

function detectSystemDark() {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

// ── 读写 ──
function load() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return { ...defaults, ...JSON.parse(raw) }
  } catch { /* ignore */ }
  return { ...defaults }
}

function save(settings) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
}

// ── 应用外观到 DOM ──
function apply(settings) {
  const root = document.documentElement
  const palette = palettes[settings.visualStyle] || palettes.aurora

  // 视觉风格色板
  root.style.setProperty('--accent', palette.accent)
  root.style.setProperty('--accent-bg', palette.accentBg)

  // 字号（通过 CSS 变量，所有组件引用此变量）
  root.style.setProperty('--font-size-base', fontSizes[settings.fontSize] || fontSizes.default)

  // 字体
  root.style.setProperty('--font-family-base', fontFamilies[settings.fontFamily] || fontFamilies.system)

  // 主题
  const effectiveTheme = settings.theme === 'auto'
    ? (detectSystemDark() ? 'dark' : 'light')
    : settings.theme
  root.setAttribute('data-theme', effectiveTheme)
}

// ── 监听系统主题变化 ──
function startAutoListener(settings) {
  if (darkModeMq) return // already listening
  darkModeMq = window.matchMedia('(prefers-color-scheme: dark)')
  darkModeMq.addEventListener('change', () => {
    const s = load()
    if (s.theme === 'auto') apply(s)
  })
}

// ── 初始化 ──
function init() {
  const settings = load()
  apply(settings)
  if (settings.theme === 'auto') startAutoListener(settings)
  return settings
}

export { defaults, palettes, fontSizes, fontFamilies, load, save, apply, init, startAutoListener }
