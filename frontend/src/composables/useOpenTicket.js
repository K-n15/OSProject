import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

export function useOpenTicket() {
  const route = useRoute()
  const router = useRouter()
  const openId = computed(() => {
    const n = Number(route.query.ticket)
    return Number.isInteger(n) && n > 0 ? n : null
  })
  const open = (id) => router.push({ query: { ...route.query, ticket: String(id) } })
  const close = () => {
    const { ticket, ...rest } = route.query
    router.replace({ query: rest })
  }
  return { openId, open, close }
}
