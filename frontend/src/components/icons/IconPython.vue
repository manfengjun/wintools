<!-- Python icon — 简化 Python 标志 -->
<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor"
       stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
    <path d="M12 2c-2.5 0-4.5.4-4.5 2v2h9V4c0-1.6-2-2-4.5-2z" />
    <path d="M7.5 8c-2.5 0-4.5 1-4.5 2.5v3c0 1.5 2 2.5 4.5 2.5H9v-2.5c0-1.5 2-2.5 4.5-2.5h3c2.5 0 4.5-1 4.5-2.5v-3C21 9 19 8 16.5 8z" />
    <path d="M16.5 16v2.5c0 1.5-2 2.5-4.5 2.5s-4.5-1-4.5-2.5V16" />
    <circle cx="9" cy="10" r=".8" fill="currentColor" stroke="none" />
    <circle cx="15" cy="18" r=".8" fill="currentColor" stroke="none" />
  </svg>
</template>
<script setup>
defineProps({ size: { type: Number, default: 22 } })
</script>
