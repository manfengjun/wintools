<script setup>
import { ref, onMounted } from 'vue'
import Sidebar from './components/Sidebar.vue'
import SettingsModal from './views/Settings.vue'
import * as theme from './theme.js'
import * as locale from './locale.js'

const showSettings = ref(false)

onMounted(() => {
  locale.init()
  theme.init()
})

function openSettings() {
  showSettings.value = true
}

function closeSettings() {
  showSettings.value = false
}

function hideToTray() {
  if (window.runtime && window.runtime.WindowHide) {
    window.runtime.WindowHide()
  } else if (window.runtime && window.runtime.WindowMinimise) {
    window.runtime.WindowMinimise()
  }
}
</script>

<template>
  <div class="app-layout">
    <Sidebar @show-settings="openSettings" @hide-to-tray="hideToTray" />
    <main class="main-content">
      <router-view />
    </main>
    <SettingsModal v-if="showSettings" @close="closeSettings" />
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
