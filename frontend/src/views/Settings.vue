<script setup>
import { ref, onMounted } from 'vue'
import * as theme from '../theme.js'
import { useT, setLocale, getLocale, detectLanguage, loadLocalePref } from '../locale.js'

const emit = defineEmits(['close'])

const t = useT()

const categories = [
  { id: 'appearance', labelKey: 'settings.appearance', subtitleKey: 'settings.appearanceSub' },
  { id: 'mirror', labelKey: 'settings.mirror', subtitleKey: 'settings.mirrorSub' },
  { id: 'language', labelKey: 'settings.language', subtitleKey: 'settings.languageSub' },
  { id: 'about', labelKey: 'settings.about', subtitleKey: 'settings.aboutSub' },
]

const activeCategory = ref('appearance')

// Theme state
const themeMode = ref('auto')
const visualStyle = ref('aurora')
const fontSize = ref('default')
const fontFamily = ref('system')
const localePref = ref('auto')

const visualStyles = [
  { id: 'graphite', tagKey: '利落',
    colors: ['#f0f0f0', '#fff', '#e8590c'],
    descZh: '纸面白配石墨文字与橙色强调，利落、克制。',
    descEn: 'Paper white with graphite text and orange accent.' },
  { id: 'aurora', tagKey: '温润',
    colors: ['#f5f5ff', '#fff', '#7c3aed'],
    descZh: '柔紫底色融合极光蓝绿，更轻盈、有呼吸感。',
    descEn: 'Soft purple with aurora blue-green, light and airy.' },
  { id: 'slate', tagKey: '原生',
    colors: ['#eef0f2', '#fff', '#2563eb'],
    descZh: '冷灰工作台配品牌蓝，发丝边框清晰。',
    descEn: 'Cool gray workbench with brand blue, crisp borders.' },
  { id: 'carbon', tagKey: '高级',
    colors: ['#d4d4d4', '#fafafa', '#0d9488'],
    descZh: '暖炭黑与米灰表面配青绿强调，适合专注。',
    descEn: 'Warm charcoal and cream with teal accent, focused.' },
]

// Mirror state
const mirrorURL = ref('https://pypi.tuna.tsinghua.edu.cn/simple')
const mirrorOptions = [
  { value: 'https://pypi.tuna.tsinghua.edu.cn/simple', labelKey: 'settings.mirrorTuna' },
  { value: 'https://mirrors.aliyun.com/pypi/simple', labelKey: 'settings.mirrorAliyun' },
  { value: 'https://pypi.org/simple', labelKey: 'settings.mirrorOfficial' },
]

onMounted(() => {
  const s = theme.load()
  themeMode.value = s.theme
  visualStyle.value = s.visualStyle
  fontSize.value = s.fontSize
  fontFamily.value = s.fontFamily
  localePref.value = loadLocalePref()
})

function onChange() {
  theme.save({
    theme: themeMode.value,
    visualStyle: visualStyle.value,
    fontSize: fontSize.value,
    fontFamily: fontFamily.value,
  })
  theme.apply({
    theme: themeMode.value,
    visualStyle: visualStyle.value,
    fontSize: fontSize.value,
    fontFamily: fontFamily.value,
  })
  // Restart auto listener if needed
  if (themeMode.value === 'auto') theme.startAutoListener()
}

function onLocaleChange() {
  if (localePref.value === 'auto') {
    setLocale(detectLanguage())
  } else {
    setLocale(localePref.value)
  }
}
</script>

<template>
  <div class="settings-overlay" @click.self="$emit('close')" role="dialog" aria-modal="true" :aria-label="t('settings.title')">
    <div class="settings-modal">
      <div class="modal-header">
        <h2 class="modal-title">{{ t('settings.title') }}</h2>
        <button class="modal-close" @click="$emit('close')" :aria-label="t('common.close')">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2" stroke-linecap="round">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
      <div class="modal-body">
        <nav class="settings-nav">
          <button v-for="cat in categories" :key="cat.id"
                  class="nav-category" :class="{ active: activeCategory === cat.id }"
                  @click="activeCategory = cat.id">
            <span class="cat-label">{{ t(cat.labelKey) }}</span>
            <span class="cat-subtitle">{{ t(cat.subtitleKey) }}</span>
          </button>
        </nav>
        <div class="settings-content">

          <!-- ═══ Appearance ═══ -->
          <div v-if="activeCategory === 'appearance'">
            <h3 class="content-title">{{ t('settings.appearance') }}</h3>
            <p class="content-subtitle">{{ t('settings.appearanceSub') }}。</p>
            <section class="setting-section">
              <div class="section-header">
                <span class="section-label">{{ t('settings.theme') }}</span>
                <div class="segmented-control">
                  <button :class="{ active: themeMode === 'auto' }" @click="themeMode='auto';onChange()">{{ t('settings.themeAuto') }}</button>
                  <button :class="{ active: themeMode === 'light' }" @click="themeMode='light';onChange()">{{ t('settings.themeLight') }}</button>
                  <button :class="{ active: themeMode === 'dark' }" @click="themeMode='dark';onChange()">{{ t('settings.themeDark') }}</button>
                </div>
              </div>
            </section>
            <section class="setting-section">
              <span class="section-label">{{ t('settings.visualStyle') }}</span>
              <div class="style-grid">
                <button v-for="vs in visualStyles" :key="vs.id"
                        class="style-card" :class="{ active: visualStyle === vs.id }"
                        @click="visualStyle=vs.id;onChange()">
                  <div class="style-preview">
                    <span v-for="(c,i) in vs.colors" :key="i" class="color-swatch"
                          :style="{ background: c, width: i===0 ? '100%' : '32px' }"></span>
                  </div>
                  <div class="style-info">
                    <span class="style-name">{{ vs.id.charAt(0).toUpperCase() + vs.id.slice(1) }}</span>
                    <span class="style-tag">{{ vs.tagKey }}</span>
                  </div>
                  <p class="style-desc">{{ getLocale() === 'en' ? vs.descEn : vs.descZh }}</p>
                  <div v-if="visualStyle === vs.id" class="check-badge">✓</div>
                </button>
              </div>
            </section>
            <section class="setting-section">
              <div class="section-header">
                <span class="section-label">{{ t('settings.fontSize') }}</span>
                <div class="segmented-control">
                  <button :class="{ active: fontSize==='small' }" @click="fontSize='small';onChange()">{{ t('settings.fontSizeSmall') }}</button>
                  <button :class="{ active: fontSize==='default' }" @click="fontSize='default';onChange()">{{ t('settings.fontSizeDefault') }}</button>
                  <button :class="{ active: fontSize==='large' }" @click="fontSize='large';onChange()">{{ t('settings.fontSizeLarge') }}</button>
                  <button :class="{ active: fontSize==='xlarge' }" @click="fontSize='xlarge';onChange()">{{ t('settings.fontSizeXLarge') }}</button>
                </div>
              </div>
            </section>
            <section class="setting-section">
              <div class="section-header">
                <span class="section-label">{{ t('settings.fontFamily') }}</span>
                <div class="segmented-control">
                  <button :class="{ active: fontFamily==='system' }" @click="fontFamily='system';onChange()">{{ t('settings.fontSystem') }}</button>
                  <button :class="{ active: fontFamily==='msyh' }" @click="fontFamily='msyh';onChange()">{{ t('settings.fontMsyh') }}</button>
                  <button :class="{ active: fontFamily==='source' }" @click="fontFamily='source';onChange()">{{ t('settings.fontSource') }}</button>
                </div>
              </div>
            </section>
          </div>

          <!-- ═══ Mirror ═══ -->
          <div v-if="activeCategory === 'mirror'">
            <h3 class="content-title">{{ t('settings.mirror') }}</h3>
            <p class="content-subtitle">{{ t('settings.mirrorDesc') }}。</p>
            <section class="setting-section">
              <span class="section-label">{{ t('settings.mirrorSection') }}</span>
              <div style="display:flex;flex-direction:column;gap:10px;margin-top:12px;">
                <label v-for="opt in mirrorOptions" :key="opt.value" class="radio-row">
                  <input type="radio" v-model="mirrorURL" :value="opt.value" class="radio-input" />
                  <span class="radio-dot" aria-hidden="true"></span>
                  <span>{{ t(opt.labelKey) }}</span>
                </label>
              </div>
              <div style="margin-top:14px;">
                <label class="input-label">{{ t('settings.mirrorCustom') }}</label>
                <input v-model="mirrorURL" class="input" placeholder="https://..." />
              </div>
            </section>
          </div>

          <!-- ═══ Language ═══ -->
          <div v-if="activeCategory === 'language'">
            <h3 class="content-title">{{ t('settings.language') }}</h3>
            <p class="content-subtitle">{{ t('settings.languageSub') }}。</p>
            <section class="setting-section">
              <div style="display:flex;flex-direction:column;gap:12px;margin-top:4px;">
                <label class="radio-row">
                  <input type="radio" v-model="localePref" value="auto" class="radio-input" @change="onLocaleChange" />
                  <span class="radio-dot" aria-hidden="true"></span>
                  <span>{{ t('settings.langAuto') }}</span>
                </label>
                <label class="radio-row">
                  <input type="radio" v-model="localePref" value="zh" class="radio-input" @change="onLocaleChange" />
                  <span class="radio-dot" aria-hidden="true"></span>
                  <span>{{ t('settings.langZh') }}</span>
                </label>
                <label class="radio-row">
                  <input type="radio" v-model="localePref" value="en" class="radio-input" @change="onLocaleChange" />
                  <span class="radio-dot" aria-hidden="true"></span>
                  <span>{{ t('settings.langEn') }}</span>
                </label>
              </div>
            </section>
          </div>

          <!-- ═══ About ═══ -->
          <div v-if="activeCategory === 'about'">
            <h3 class="content-title">{{ t('settings.about') }}</h3>
            <p class="content-subtitle">{{ t('about.title') }} · {{ t('about.version') }}</p>
            <section class="setting-section">
              <div style="line-height:2;">
                <div><strong>{{ t('app.name') }}</strong> <span class="version-tag">{{ t('about.version') }}</span></div>
                <div style="color:var(--text-muted);font-size:13px;">{{ t('app.tagline') }}</div>
                <div style="color:var(--text-muted);font-size:13px;margin-top:8px;">{{ t('settings.aboutBuiltWith') }}</div>
              </div>
            </section>
          </div>

        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-overlay {
  position: fixed; inset: 0;
  background: var(--overlay-bg);
  display: flex; align-items: center; justify-content: center;
  z-index: 200; animation: fadeIn 0.15s ease;
}
.settings-modal {
  width: 720px; max-width: 92vw; height: 520px; max-height: 85vh;
  background: var(--bg-elevated);
  border-radius: 16px;
  box-shadow: var(--shadow-lg);
  display: flex; flex-direction: column;
  overflow: hidden;
  animation: modalIn 0.2s ease;
}
@keyframes modalIn {
  from { opacity: 0; transform: scale(0.96) translateY(8px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}
.modal-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 18px 24px; border-bottom: 1px solid var(--border-default); flex-shrink: 0;
}
.modal-title { font-size: 1.14rem; font-weight: 700; color: var(--text-primary); }
.modal-close {
  display: flex; align-items: center; justify-content: center;
  width: 32px; height: 32px; border: none; border-radius: 6px;
  background: transparent; color: var(--text-muted); cursor: pointer; transition: all 0.15s;
}
.modal-close:hover { background: var(--bg-nav-hover); color: var(--text-primary); }
.modal-body { display: flex; flex: 1; overflow: hidden; }

.settings-nav {
  width: 180px; min-width: 180px;
  padding: 12px 8px; border-right: 1px solid var(--border-default);
  display: flex; flex-direction: column; gap: 2px;
}
.nav-category {
  display: flex; flex-direction: column; align-items: flex-start; gap: 1px;
  padding: 10px 14px; border: none; border-radius: 6px;
  background: transparent; cursor: pointer; text-align: left;
  transition: all 0.15s; border-left: 3px solid transparent;
}
.nav-category:hover { background: var(--bg-nav-hover); }
.nav-category.active { background: var(--accent-bg); border-left-color: var(--accent); }
.cat-label { font-size: 1rem; font-weight: 500; color: var(--text-primary); }
.nav-category.active .cat-label { color: var(--accent); }
.cat-subtitle { font-size: 0.79rem; color: var(--text-placeholder); line-height: 1.3; }
.nav-category.active .cat-subtitle { color: var(--text-muted); }

.settings-content { flex: 1; padding: 24px 28px; overflow-y: auto; }
.content-title { font-size: 1.14rem; font-weight: 700; color: var(--text-primary); margin-bottom: 2px; }
.content-subtitle { font-size: 0.93rem; color: var(--text-muted); margin-bottom: 24px; }
.setting-section { margin-bottom: 24px; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.section-label { font-size: 0.93rem; font-weight: 600; color: var(--text-secondary); display: block; margin-bottom: 8px; }

.segmented-control {
  display: flex; background: var(--bg-segmented); border-radius: 8px; padding: 2px; gap: 2px;
}
.segmented-control button {
  padding: 6px 14px; border: none; border-radius: 6px;
  background: transparent; font-size: 0.93rem; color: var(--text-secondary);
  cursor: pointer; transition: all 0.15s; white-space: nowrap;
}
.segmented-control button.active {
  background: var(--bg-segmented-active); color: var(--color-primary); font-weight: 500;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
}
.segmented-control button:hover:not(.active) { color: var(--text-primary); }

.style-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; margin-top: 12px; }
.style-card {
  position: relative; border: 1.5px solid var(--border-default); border-radius: 10px;
  padding: 14px; background: var(--bg-surface); cursor: pointer; text-align: left; transition: all 0.15s;
}
.style-card:hover { border-color: var(--border-input-hover); }
.style-card.active { border-color: var(--accent); background: var(--accent-bg); }
.style-preview { display: flex; height: 36px; border-radius: 6px; overflow: hidden; margin-bottom: 10px; gap: 2px; }
.color-swatch { display: block; border-radius: 4px; }
.style-info { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.style-name { font-size: 0.93rem; font-weight: 600; color: var(--text-primary); }
.style-tag { font-size: 0.79rem; color: var(--text-muted); background: var(--bg-code); padding: 1px 8px; border-radius: 4px; }
.style-desc { font-size: 0.86rem; color: var(--text-muted); line-height: 1.4; }
.check-badge {
  position: absolute; top: 8px; right: 8px;
  width: 22px; height: 22px; border-radius: 50%; background: var(--accent);
  color: #fff; display: flex; align-items: center; justify-content: center;
  font-size: 12px; font-weight: 700;
}

.radio-row { display: flex; align-items: center; gap: 10px; cursor: pointer; font-size: 1rem; padding: 4px 0; color: var(--text-secondary); }
.radio-input { position: absolute; opacity: 0; width: 0; height: 0; }
.radio-dot {
  width: 18px; height: 18px; border-radius: 50%; border: 2px solid var(--border-input);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0; transition: all 0.15s;
}
.radio-dot::after {
  content: ''; width: 8px; height: 8px; border-radius: 50%;
  background: var(--color-primary); transform: scale(0); transition: transform 0.15s;
}
.radio-input:checked + .radio-dot { border-color: var(--color-primary); }
.radio-input:checked + .radio-dot::after { transform: scale(1); }
.input-label { display: block; font-size: 0.86rem; color: var(--text-muted); margin-bottom: 6px; }

.version-tag {
  display: inline-block; font-size: 0.79rem; color: #fff;
  background: var(--color-primary); padding: 1px 8px; border-radius: 4px;
  vertical-align: middle; margin-left: 6px;
}
</style>
