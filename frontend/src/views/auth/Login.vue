<script setup>
import { ref, reactive, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuthCard from '../../components/AuthCard.vue'
import FormField from '../../components/FormField.vue'
import { login, homePath } from '../../session.js'

const route = useRoute()
const router = useRouter()

const NOTICES = {
  expired: 'Your session ended. Please log in again.',
  logout: 'You have logged out.',
}
const notice = computed(() => NOTICES[route.query.msg] || '')

const f = reactive({ username: '', password: '' })
const formError = ref('')
const busy = ref(false)

async function submit() {
  formError.value = ''
  if (!f.username.trim() || !f.password) {
    formError.value = 'Enter your username and password.'
    return
  }
  busy.value = true
  try {
    const user = await login(f.username.trim(), f.password)
    const next = String(route.query.next || '')
    router.replace(next.startsWith('/') && !next.startsWith('//') ? next : homePath(user))
  } catch (e) {
    formError.value = e.message
    f.password = ''
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthCard title="Login">
    <p v-if="notice" class="notice" role="status">{{ notice }}</p>
    <form class="auth-form" novalidate @submit.prevent="submit">
      <FormField v-model="f.username" label="Username" autocomplete="username" autofocus />
      <FormField v-model="f.password" label="Password" type="password" autocomplete="current-password" />
      <p v-if="formError" class="error" role="alert">{{ formError }}</p>
      <button class="btn block" :disabled="busy">{{ busy ? 'Logging in…' : 'Login' }}</button>
    </form>
    <p class="auth-lead small">Accounts are created by the IT team. Ask them if you need one.</p>
  </AuthCard>
</template>
