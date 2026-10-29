import { ref, computed, watch, onUnmounted } from 'vue'
import { session } from '../session'

export function usePagedList(fetchPage, filters, pageSize) {
  const page = ref(1)
  const query = ref('')
  const items = ref([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref('')
  const pages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
  const empty = computed(() => !loading.value && !error.value && !items.value.length)

  let seq = 0
  async function load({ background = false } = {}) {
    const mine = ++seq
    if (!background) {
      loading.value = true
      error.value = ''
    }
    try {
      const res = await fetchPage({ ...filters(), q: query.value.trim(), page: page.value, pageSize })
      if (mine !== seq) return
      items.value = res.items
      total.value = res.total
      error.value = ''
      if (page.value > pages.value) page.value = pages.value
    } catch (e) {
      if (mine === seq && !background) error.value = e.message
    } finally {
      if (mine === seq) loading.value = false
    }
  }

  const restart = () => (page.value === 1 ? load() : (page.value = 1))

  let timer
  watch(query, () => {
    clearTimeout(timer)
    timer = setTimeout(restart, 300)
  })
  watch(filters, restart)
  watch(page, () => load())
  watch(() => session.version, () => load())
  const poll = setInterval(() => {
    if (document.visibilityState === 'visible') load({ background: true })
  }, 15000)
  onUnmounted(() => {
    clearTimeout(timer)
    clearInterval(poll)
  })
  load()

  return { page, query, items, total, pages, loading, error, empty, pageSize, reload: () => load() }
}
