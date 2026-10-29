<script setup>
import Icon from './Icon.vue'
const query = defineModel('query', { default: '' })
const page = defineModel('page', { default: 1 })
defineProps({ pages: { type: Number, default: 1 } })
</script>
<template>
  <div class="toolbar">
    <slot name="filters" />
    <label class="search">
      <Icon name="search" :size="16" />
      <input v-model="query" type="search" placeholder="Search issues" aria-label="Search issues" />
    </label>
    <slot />
    <div class="pager">
      <span>{{ page }} of {{ pages }}</span>
      <button class="icon-btn" :disabled="page <= 1" @click="page--" aria-label="Previous page"><Icon name="left" :size="16" /></button>
      <button class="icon-btn" :disabled="page >= pages" @click="page++" aria-label="Next page"><Icon name="right" :size="16" /></button>
    </div>
  </div>
</template>
