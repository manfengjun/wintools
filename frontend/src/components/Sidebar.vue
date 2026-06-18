<script setup>
import { useRoute } from 'vue-router'
import tools from '../tools.js'
import { useT } from '../locale.js'

const emit = defineEmits(['show-settings', 'hide-to-tray'])
const route = useRoute()
const t = useT()

function isActive(path) {
  return route.path === '/' + path
}
</script>

<template>
  <nav class="sidebar" role="navigation" :aria-label="t('nav.settings')">
    <div class="sidebar-header">
      <div class="logo" aria-hidden="true">
        <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="2" y="3" width="20" height="14" rx="2" />
          <path d="M8 21h8" />
          <path d="M12 17v4" />
        </svg>
      </div>
      <span class="title">{{ t('app.name') }}</span>
    </div>

    <div class="nav-items" role="tablist">
      <router-link
        v-for="tool in tools"
        :key="tool.id"
        :to="'/' + tool.id"
        class="nav-item"
        :class="{ active: isActive(tool.id) }"
        :aria-label="t(tool.labelKey)"
        :aria-current="isActive(tool.id) ? 'page' : undefined"
        role="tab"
      >
        <component :is="tool.icon" class="nav-icon" aria-hidden="true" />
        <span class="nav-label">{{ t(tool.labelKey) }}</span>
      </router-link>
    </div>

    <div class="sidebar-footer">
      <button class="footer-btn" @click="$emit('show-settings')" :aria-label="t('nav.settings')" :title="t('nav.settings')">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="3"/>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
        </svg>
      </button>
      <button class="footer-btn" @click="$emit('hide-to-tray')" :aria-label="t('nav.minimize')" :title="t('nav.minimize')">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2" stroke-linecap="round">
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
      </button>
    </div>
  </nav>
</template>

<style scoped>
.sidebar {
  width: 200px;
  min-width: 200px;
  background: var(--bg-surface);
  border-right: 1px solid var(--sidebar-border);
  display: flex;
  flex-direction: column;
  user-select: none;
  transition: background var(--transition-normal), border-color var(--transition-normal);
}
.sidebar-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px 16px 16px;
  border-bottom: 1px solid var(--sidebar-header-border);
}
.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: linear-gradient(135deg, var(--accent), oklch(from var(--accent) 0.6 0.08 h));
  color: var(--text-on-accent);
}
.title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}
.nav-items {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 12px 8px;
  overflow-y: auto;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  text-decoration: none;
  color: var(--text-secondary);
  font-size: 14px;
  transition: all var(--transition-fast);
  outline: none;
  cursor: pointer;
}
.nav-item:hover { background: var(--bg-nav-hover); color: var(--text-primary); }
.nav-item:focus-visible { box-shadow: 0 0 0 2px var(--color-primary); }
.nav-item.active {
  background: var(--accent-bg);
  color: var(--accent);
  font-weight: 500;
}
.nav-icon { width: 22px; height: 22px; flex-shrink: 0; }
.nav-label { font-size: 13px; line-height: 1; }
.sidebar-footer {
  display: flex;
  gap: 2px;
  padding: 10px 8px;
  border-top: 1px solid var(--sidebar-header-border);
  justify-content: center;
}
.footer-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.footer-btn:hover { background: var(--bg-nav-hover); color: var(--text-secondary); }
.footer-btn:focus-visible { box-shadow: 0 0 0 2px var(--color-primary); }
</style>
