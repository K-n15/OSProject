<script setup>
import { ref, computed, watch } from 'vue'
import ListToolbar from '../components/ListToolbar.vue'
import TicketDetail from '../components/TicketDetail.vue'
import Icon from '../components/Icon.vue'
import { countTickets, listTickets } from '../api'
import { session } from '../session'
import { usePagedList } from '../composables/usePagedList'
import { useOpenTicket } from '../composables/useOpenTicket'
import { fmt, MAX_ONGOING } from '../format'

const props = defineProps({ status: { type: String, required: true } })

const list = usePagedList(listTickets, () => ({ status: props.status }), 10)
const { page, query, items, pages, loading, error, empty, pageSize } = list
const { openId, open, close } = useOpenTicket()

const ongoing = ref(0)
async function refreshCapacity() {
  try { ongoing.value = await countTickets('ongoing') } catch { /* shown elsewhere */ }
}
watch(() => session.version, refreshCapacity, { immediate: true })

const cols = computed(() => (props.status === 'closed' ? 5 : 4))
</script>

<template>
  <ListToolbar v-model:query="query" v-model:page="page" :pages="pages">
    <span v-if="status !== 'closed'" class="capacity" :class="{ full: ongoing >= MAX_ONGOING }">
      {{ ongoing }} of {{ MAX_ONGOING }} ongoing slots in use
    </span>
  </ListToolbar>

  <p v-if="error" class="error banner" role="alert">{{ error }} <button class="btn ghost" @click="list.reload">Retry</button></p>

  <table class="grid" :class="{ busy: loading }">
    <thead>
      <tr>
        <th class="num">No.</th><th>Issue</th><th>Reported</th>
        <th v-if="status === 'closed'">Closed</th><th class="chev"></th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="(t, i) in items" :key="t.id">
        <td class="num">{{ (page - 1) * pageSize + i + 1 }}.</td>
        <td><button class="link" @click="open(t.id)">{{ t.issue }}</button></td>
        <td>{{ fmt(t.reported) }}</td>
        <td v-if="status === 'closed'">{{ fmt(t.closed) }}</td>
        <td class="chev">
          <button class="icon-btn" @click="open(t.id)" aria-label="View details"><Icon name="right" :size="18" /></button>
        </td>
      </tr>
      <tr v-if="empty"><td :colspan="cols" class="empty">No {{ status }} tickets.</td></tr>
    </tbody>
  </table>

  <TicketDetail v-if="openId" :key="openId" :id="openId" @close="close" />
</template>
