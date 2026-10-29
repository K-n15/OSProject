<script setup>
import { ref, computed, onUnmounted } from 'vue'
import ListToolbar from '../components/ListToolbar.vue'
import TicketDetail from '../components/TicketDetail.vue'
import { createTicket, listTickets } from '../api'
import { session, ticketsChanged } from '../session'
import { usePagedList } from '../composables/usePagedList'
import { useOpenTicket } from '../composables/useOpenTicket'
import { fmt } from '../format'

const status = ref('all')
const list = usePagedList(listTickets, () => ({ status: status.value === 'all' ? '' : status.value }), 8)
const { page, query, items, pages, loading, error, empty, pageSize } = list
const { openId, open, close } = useOpenTicket()

// New ticket form
const MAX_IMAGE = 2 * 1024 * 1024
const showNew = ref(false)
const form = ref({ title: '', deviceId: '', description: '', image: null })
const previewUrl = ref('')
const formError = ref('')
const submitting = ref(false)

function setPreview(file) {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = file ? URL.createObjectURL(file) : ''
}
onUnmounted(() => setPreview(null))

function pickImage(e) {
  formError.value = ''
  const file = e.target.files[0] || null
  form.value.image = null
  setPreview(null)
  if (!file) return
  if (!file.type.startsWith('image/')) return (formError.value = 'Choose an image file (PNG, JPEG, GIF or WebP).')
  if (file.size > MAX_IMAGE) return (formError.value = 'Image must be under 2 MB.')
  form.value.image = file // the real File is uploaded; the server checks it again
  setPreview(file)
}

function openForm() {
  form.value = { title: '', deviceId: '', description: '', image: null }
  formError.value = ''
  setPreview(null)
  showNew.value = true
}

async function submit() {
  submitting.value = true
  formError.value = ''
  try {
    await createTicket({ ...form.value, title: form.value.title.trim(), description: form.value.description.trim() })
    showNew.value = false
    setPreview(null)
    status.value = 'all'
    ticketsChanged()
  } catch (e) {
    formError.value = e.message
  } finally {
    submitting.value = false
  }
}

const lastUpdate = (t) => (t.updated === t.reported ? fmt(null) : fmt(t.updated))
</script>

<template>
  <ListToolbar v-model:query="query" v-model:page="page" :pages="pages">
    <template #filters>
      <select v-model="status" aria-label="Filter by status">
        <option value="all">All statuses</option>
        <option value="pending">Pending</option>
        <option value="ongoing">Ongoing</option>
        <option value="closed">Closed</option>
      </select>
    </template>
    <button class="btn" @click="openForm">New</button>
  </ListToolbar>

  <p v-if="error" class="error banner" role="alert">{{ error }} <button class="btn ghost" @click="list.reload">Retry</button></p>

  <table class="grid" :class="{ busy: loading }">
    <thead><tr><th class="num">No.</th><th>Issue</th><th>Status</th><th>Reported</th><th>Last update</th></tr></thead>
    <tbody>
      <tr v-for="(t, i) in items" :key="t.id">
        <td class="num">{{ (page - 1) * pageSize + i + 1 }}.</td>
        <td><button class="link" @click="open(t.id)">{{ t.issue }}</button></td>
        <td><span class="status" :class="t.status">{{ t.status }}</span></td>
        <td>{{ fmt(t.reported) }}</td>
        <td>{{ lastUpdate(t) }}</td>
      </tr>
      <tr v-if="empty"><td colspan="5" class="empty">No tickets found. Select New to report an issue.</td></tr>
    </tbody>
  </table>

  <TicketDetail v-if="openId" :key="openId" :id="openId" @close="close" />

  <div v-if="showNew" class="overlay" @click.self="showNew = false">
    <form class="dialog" @submit.prevent="submit">
      <h2>New ticket</h2>
      <label>Ticket header<input v-model="form.title" required maxlength="80" placeholder="e.g. Unable to connect printer" /></label>
      <label>Device
        <select v-model.number="form.deviceId" required>
          <option value="" disabled>Select a device</option>
          <option v-for="d in session.devices" :key="d.id" :value="d.id">{{ d.name }}</option>
        </select>
      </label>
      <label>Description<textarea v-model="form.description" required maxlength="5000" rows="4" placeholder="What happened, and when did it start?"></textarea></label>
      <label>Image (optional)<input type="file" accept="image/png,image/jpeg,image/gif,image/webp" @change="pickImage" /></label>
      <img v-if="previewUrl" :src="previewUrl" alt="Selected attachment preview" class="preview" />
      <p v-if="formError" class="error" role="alert">{{ formError }}</p>
      <div class="dialog-actions">
        <button type="button" class="btn ghost" @click="showNew = false">Cancel</button>
        <button class="btn" :disabled="submitting">{{ submitting ? 'Submitting…' : 'Submit ticket' }}</button>
      </div>
    </form>
  </div>
</template>
