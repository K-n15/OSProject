<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { getTicket, sendReply, acceptTicket, closeTicket, countTickets, markNotificationsRead } from '../api'
import { session, ticketsChanged } from '../session'
import { fmtTime, MAX_ONGOING } from '../format'

const props = defineProps({ id: { type: Number, required: true } })
const emit = defineEmits(['close'])

const t = ref(null)
const loadError = ref('')
const actionError = ref('')
const busy = ref(false)
const text = ref('')
const resolveTypeId = ref('')
const resolveNote = ref('')
const ongoing = ref(0)

const me = computed(() => session.user)
const isSupport = computed(() => me.value.role !== 'employee')
const mine = computed(() => t.value?.assigneeId === me.value.id)

const canReply = computed(() =>
  t.value && t.value.status !== 'closed' && (!isSupport.value || (t.value.status === 'ongoing' && mine.value)))

const thread = ref(null)

async function load({ background = false } = {}) {
  if (!background) loadError.value = ''
  try {
    const fresh = await getTicket(props.id)
    const before = t.value
    const changed = !before || before.updated !== fresh.updated
    const newMessages = before && (fresh.messages?.length || 0) > (before.messages?.length || 0)
    t.value = fresh
    if (isSupport.value && fresh.status === 'pending') ongoing.value = await countTickets('ongoing')
    if (changed) seen()
    if (newMessages) {
      await nextTick()
      thread.value?.lastElementChild?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
  } catch (e) {
    if (!background) loadError.value = e.message
  }
}

function seen() {
  markNotificationsRead(props.id)
    .then(() => session.notifVersion++)
    .catch(() => { /* not worth an error message */ })
}

let timer
onMounted(() => {
  load()
  timer = setInterval(() => {
    if (!busy.value && document.visibilityState === 'visible') load({ background: true })
  }, 5000)
})
onUnmounted(() => clearInterval(timer))

async function run(action, { closeAfter = false } = {}) {
  busy.value = true
  actionError.value = ''
  try {
    await action()
    ticketsChanged()
    if (closeAfter) emit('close')
    else await load()
  } catch (e) {
    actionError.value = e.message
    await load()
  } finally {
    busy.value = false
  }
}

const send = () => run(async () => { await sendReply(props.id, text.value.trim()); text.value = '' })
const accept = () => run(() => acceptTicket(props.id), { closeAfter: true })
const close = () => run(() => closeTicket(props.id, resolveTypeId.value, resolveNote.value.trim()), { closeAfter: true })
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <section class="dialog wide" role="dialog" aria-modal="true" :aria-label="t ? t.issue : 'Ticket'">
      <p v-if="loadError" class="error" role="alert">{{ loadError }}</p>
      <p v-else-if="!t" class="hint">Loading ticket…</p>

      <template v-else>
        <header class="dialog-head">
          <h2>{{ t.issue }}</h2>
          <span class="status" :class="t.status">{{ t.status }}</span>
          <button class="btn ghost" @click="emit('close')">Close window</button>
        </header>

        <dl class="meta">
          <div><dt>Device</dt><dd>{{ t.device }}</dd></div>
          <div><dt>Submitted</dt><dd>{{ fmtTime(t.reported) }}</dd></div>
          <div><dt>Last update</dt><dd>{{ fmtTime(t.updated) }}</dd></div>
          <template v-if="t.status === 'closed'">
            <div><dt>Closed</dt><dd>{{ fmtTime(t.closed) }}</dd></div>
            <div><dt>Resolve type</dt><dd>{{ t.resolveType }}</dd></div>
          </template>
        </dl>

        <div ref="thread" class="thread" aria-live="polite">
          <div class="msg employee">
            <p>{{ t.description }}</p>
            <img v-for="a in t.attachments || []" :key="a.id" :src="a.url" alt="Attachment from employee" />
            <small>Employee, {{ fmtTime(t.reported) }}</small>
          </div>
          <div v-for="m in t.messages || []" :key="m.id" class="msg" :class="m.from">
            <p>{{ m.text }}</p>
            <small>{{ m.from === 'support' ? 'IT support' : 'Employee' }}, {{ fmtTime(m.at) }}</small>
          </div>
          <div v-if="t.status === 'closed'" class="resolution">
            <strong>Resolved: {{ t.resolveType }}</strong>
            <p v-if="t.resolveNote">{{ t.resolveNote }}</p>
          </div>
        </div>

        <p v-if="actionError" class="error" role="alert">{{ actionError }}</p>

        <form v-if="canReply" class="reply" @submit.prevent="send">
          <textarea v-model="text" rows="2" maxlength="5000" aria-label="Reply"
            :placeholder="isSupport ? 'Reply to the employee' : 'Add a reply for IT support'"></textarea>
          <button class="btn" :disabled="busy || !text.trim()">Send reply</button>
        </form>
        <p v-else-if="t.status === 'pending' && isSupport" class="hint">Accept this ticket to reply to the employee.</p>
        <p v-else-if="t.status === 'ongoing' && isSupport && !mine" class="hint">Another IT member is handling this ticket.</p>
        <p v-else-if="t.status === 'closed'" class="hint">This ticket is closed, so replies are turned off.</p>

        <div v-if="isSupport && t.status === 'pending'" class="actions">
          <button class="btn" :disabled="busy || ongoing >= MAX_ONGOING" @click="accept">Accept ticket</button>
          <span v-if="ongoing >= MAX_ONGOING" class="hint">
            You have {{ ongoing }} of {{ MAX_ONGOING }} ongoing tickets. Close one before accepting another.
          </span>
        </div>

        <form v-if="isSupport && t.status === 'ongoing' && mine" class="close-form" @submit.prevent="close">
          <h3>Close ticket</h3>
          <label>Resolve type
            <select v-model.number="resolveTypeId" required>
              <option value="" disabled>Select a resolve type</option>
              <option v-for="r in session.resolveTypes" :key="r.id" :value="r.id">{{ r.name }}</option>
            </select>
          </label>
          <label>Description
            <textarea v-model="resolveNote" rows="3" maxlength="5000" placeholder="What was done to resolve it?"></textarea>
          </label>
          <button class="btn" :disabled="busy || !resolveTypeId">Close ticket</button>
        </form>
      </template>
    </section>
  </div>
</template>
