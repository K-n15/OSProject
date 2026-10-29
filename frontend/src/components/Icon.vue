<script setup>
defineProps({ name: String, size: { type: Number, default: 20 } })
const paths = {
  ticket: 'M3 8a2 2 0 0 0 0 8v2h18v-2a2 2 0 0 1 0-8V6H3zM13 6v12',
  book: 'M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2zM4 19V5M9 8h6',
  help: 'M9.5 9a2.5 2.5 0 1 1 3.5 2.3c-.7.4-1 .9-1 1.7M12 17h.01',
  bell: 'M6 16V11a6 6 0 1 1 12 0v5l2 2H4zM10 21h4',
  search: 'M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM21 21l-5-5',
  play: 'M7 4l13 8-13 8z',
  check: 'M4 12l5 5L20 6',
  down: 'M6 9l6 6 6-6',
  left: 'M15 6l-6 6 6 6',
  right: 'M9 6l6 6-6 6',
  logout: 'M9 4H5v16h4M14 8l4 4-4 4M18 12H8',
}
</script>
<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path :d="paths[name]" />
    <circle v-if="name === 'help'" cx="12" cy="12" r="9" />
  </svg>
</template>
