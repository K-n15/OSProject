<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import Icon from './Icon.vue'
import { getNotifications, markNotificationsRead } from '../api.js'
import { session, roleArea } from '../session.js'

const router = useRouter()
const open = ref(false)
const items = ref([])
const unread = ref(0)
const error = ref('')
const root = ref(null)
const bell = ref(null)

async function refresh() {
  if (!session.user) return
  try {
    const res = await getNotifications()
    items.value = res.items
    unread.value = res.unread
    error.value = ''
  } catch (e) {
    error.value = e.message
  }
}

let timer
onMounted(() => {
  refresh()
  timer = setInterval(refresh, 15000)
})
onUnmounted(() => clearInterval(timer))
watch(() => [session.version, session.notifVersion], refresh)

function toggle() {
  open.value = !open.value
  if (open.value) refresh()
}

function onPointerDown(e) {
  if (open.value && root.value && !root.value.contains(e.target)) open.value = false
}
function onKey(e) {
  if (e.key === 'Escape' && open.value) {
    open.value = false
    nextTick(() => bell.value?.focus())
  }
}
onMounted(() => {
  document.addEventListener('pointerdown', onPointerDown)
  document.addEventListener('keydown', onKey)
})
onUnmounted(() => {
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKey)
})

const TEXT = {
  new_ticket: (n) => `${n.actorName} opened a new ticket`,
  accepted: (n) => `${n.actorName} accepted your ticket`,
  reply: (n) => `${n.actorName} replied`,
  closed: (n) => `${n.actorName} closed your ticket`,
}
const sentence = (n) => (TEXT[n.kind] || (() => 'Ticket updated'))(n)

function ago(iso) {
  const s = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000)
  if (s < 60) return 'just now'
  if (s < 3600) return `${Math.floor(s / 60)} min ago`
  if (s < 86400) return `${Math.floor(s / 3600)} h ago`
  return new Date(iso).toLocaleDateString([], { month: 'short', day: 'numeric' })
}

function ticketLink(n) {
  if (roleArea(session.user) === 'employee') return { path: '/employee/tickets', query: { ticket: String(n.ticketId) } }
  return { path: `/support/${n.ticketStatus}`, query: { ticket: String(n.ticketId) } }
}

async function go(n) {
  open.value = false
  router.push(ticketLink(n))
  items.value = items.value.map((x) => (x.ticketId === n.ticketId ? { ...x, read: true } : x))
}

async function markAll() {
  try {
    await markNotificationsRead()
    items.value = items.value.map((x) => ({ ...x, read: true }))
    unread.value = 0
  } catch (e) {
    error.value = e.message
  }
}

const label = computed(() => (unread.value ? `Notifications, ${unread.value} unread` : 'Notifications'))
</script>

<template>
  <div ref="root" class="notif">
    <button ref="bell" class="icon-btn bell" :aria-label="label" :aria-expanded="open" aria-controls="notif-panel" @click="toggle">
      <Icon name="bell" />
      <span v-if="unread" class="badge">{{ unread > 99 ? '99+' : unread }}</span>
    </button>

    <div v-if="open" id="notif-panel" class="notif-panel" role="region" aria-label="Notifications">
      <header class="notif-head">
        <h2>Notifications</h2>
        <button v-if="unread" class="text-link" @click="markAll">Mark all as read</button>
      </header>
      <p v-if="error" class="error notif-msg" role="alert">{{ error }}</p>
      <p v-else-if="!items.length" class="notif-msg">No updates yet. You'll see ticket activity here.</p>
      <ul v-else class="notif-list">
        <li v-for="n in items" :key="n.id">
          <button class="notif-item" :class="{ unread: !n.read }" @click="go(n)">
            <span class="dot" :aria-label="n.read ? undefined : 'Unread'"></span>
            <span class="notif-text">
              <span class="notif-line">{{ sentence(n) }}</span>
              <span class="notif-title">{{ n.ticketTitle }}</span>
              <small>{{ ago(n.createdAt) }}</small>
            </span>
          </button>
        </li>
      </ul>
    </div>
  </div>
</template>
