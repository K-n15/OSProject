<script setup>
import { computed, watch, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from './components/Icon.vue'
import NotificationBell from './components/NotificationBell.vue'
import { session, logout, roleArea, pingApi } from './session'

const route = useRoute()
const router = useRouter()

const navs = {
  employee: [
    { to: '/employee/tickets', label: 'Tickets', icon: 'ticket' },
    { to: '/employee/solutions', label: 'Solutions', icon: 'book' },
    { to: '/employee/faq', label: 'FAQ', icon: 'help' },
  ],
  support: [
    { to: '/support/pending', label: 'Pending', icon: 'ticket' },
    { to: '/support/ongoing', label: 'Ongoing', icon: 'play' },
    { to: '/support/closed', label: 'Closed', icon: 'check' },
  ],
}
const area = computed(() => roleArea(session.user))
const nav = computed(() => navs[area.value])
const initials = computed(() => (session.user?.username || '?').slice(0, 2).toUpperCase())
const showShell = computed(() => session.user && !route.meta.public && route.name !== 'root')

const reload = () => window.location.reload()

let retryTimer
watch(() => session.error, (err) => {
  clearInterval(retryTimer)
  if (!err) return
  retryTimer = setInterval(async () => {
    try {
      await pingApi()
      reload()
    } catch { /* still down */ }
  }, 5000)
}, { immediate: true })
onUnmounted(() => clearInterval(retryTimer))

async function doLogout() {
  await logout()
  router.replace({ path: '/login', query: { msg: 'logout' } })
}
</script>

<template>
  <div v-if="!session.checked" class="center-message">Loading…</div>

  <div v-else-if="session.error" class="center-message" role="alert">
    <h1>IT-Assist is not available</h1>
    <p>{{ session.error }}</p>
    <ul v-if="session.errorReason === 'unreachable'" class="checklist">
      <li>Docker: <code>docker compose ps</code> should list <code>api</code> as running; see why with <code>docker compose logs api</code>.</li>
      <li>Development: the Go API must be running (<code>go run ./cmd/api</code>) on port 8080.</li>
    </ul>
    <ul v-else-if="session.errorReason === 'wrong-api'" class="checklist">
      <li>Stop the other backend (<code>docker ps</code> shows what is running), then start the current one.</li>
    </ul>
    <p class="muted">Retrying automatically every 5 seconds…</p>
    <button class="btn" @click="reload">Try again now</button>
  </div>

  <div v-else-if="showShell" class="window">
    <header class="topbar">
      <h1>{{ route.meta.title }}</h1>
      <div class="topbar-actions">
        <NotificationBell />
        <span class="avatar" :title="`${session.user.username} (${session.user.role})`" aria-hidden="true">{{ initials }}</span>
        <span class="who">{{ session.user.username }}</span>
        <button class="icon-btn" title="Log out" aria-label="Log out" @click="doLogout"><Icon name="logout" /></button>
      </div>
    </header>
    <nav class="sidebar" aria-label="Main">
      <RouterLink v-for="n in nav" :key="n.to" :to="n.to">
        <Icon :name="n.icon" :size="22" />
        <span>{{ n.label }}</span>
      </RouterLink>
    </nav>
    <main class="content"><RouterView :key="session.user.id" /></main>
  </div>

  <div v-else-if="route.meta.public" class="auth-screen"><RouterView /></div>
</template>
