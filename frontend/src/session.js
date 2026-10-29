import { reactive } from 'vue'
import { getMe, getDevices, getResolveTypes, login as apiLogin, logout as apiLogout } from './api'

export const session = reactive({
  checked: false,
  user: null,
  devices: [],
  resolveTypes: [],
  error: '',
  errorReason: '',
  version: 0,
  notifVersion: 0,
})

export const ticketsChanged = () => session.version++

const authChannel = typeof BroadcastChannel !== 'undefined' ? new BroadcastChannel('itassist-auth') : null
if (authChannel) authChannel.onmessage = () => window.location.reload()
const announceAuthChange = () => authChannel?.postMessage('changed')

if (typeof document !== 'undefined') {
  document.addEventListener('visibilitychange', async () => {
    if (document.visibilityState !== 'visible' || !session.checked || session.error) return
    try {
      const current = await pingApi()
      if ((current?.id ?? null) !== (session.user?.id ?? null)) window.location.reload()
    } catch { /* API unreachable: the next request will show it */ }
  })
}
export const roleArea = (user) => (user?.role === 'support' ? 'support' : 'employee')
export const homePath = (user) => (roleArea(user) === 'support' ? '/support/pending' : '/employee/tickets')

async function loadLookups() {
  const [devices, resolveTypes] = await Promise.all([getDevices(), getResolveTypes()])
  Object.assign(session, { devices, resolveTypes })
}

export async function pingApi() {
  try {
    return await getMe()
  } catch (e) {
    if (e.status === 401) return null
    throw e
  }
}

export async function checkSession() {
  session.error = ''
  session.errorReason = ''
  try {
    session.user = await pingApi()
    if (session.user) await loadLookups()
  } catch (e) {
    session.user = null
    session.error = e.message
    session.errorReason = e.data?.reason || ''
  } finally {
    session.checked = true
  }
}

export async function login(username, password) {
  const user = await apiLogin(username, password)
  try {
    session.user = user
    await loadLookups()
  } catch (e) {
    clearUser()
    throw e
  }
  announceAuthChange()
  return user
}

export async function logout() {
  try { await apiLogout() } catch { /* the cookie may already be gone */ }
  clearUser()
  announceAuthChange()
}

export function clearUser() {
  session.user = null
  session.devices = []
  session.resolveTypes = []
}
