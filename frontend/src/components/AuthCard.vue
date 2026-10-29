<script setup>
import { useRouter } from 'vue-router'
import Icon from './Icon.vue'

const props = defineProps({
  title: { type: String, required: true },
  back: { type: [String, Object, Boolean], default: false },
  loginLink: { type: Boolean, default: false },
})
const emit = defineEmits(['back'])
const router = useRouter()

function goBack() {
  if (props.back === true) emit('back')
  else router.push(props.back)
}
</script>

<template>
  <main class="auth-card">
    <nav v-if="back || loginLink" class="auth-nav" aria-label="Page">
      <button v-if="back" type="button" class="text-link" @click="goBack"><Icon name="left" :size="16" /> Back</button>
      <RouterLink v-if="loginLink" to="/login" class="text-link end">Login <Icon name="right" :size="16" /></RouterLink>
    </nav>
    <p class="auth-brand">IT-Assist</p>
    <h1>{{ title }}</h1>
    <slot />
  </main>
</template>
