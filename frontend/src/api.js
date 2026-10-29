// Every call to the Go backend goes through this file. See API.md in the project root.

export class ApiError extends Error {
  constructor(status, message, data = null) {
    super(message)
    this.status = status
    this.data = data
  }
}

let onUnauthorized = () => {}
export const setUnauthorizedHandler = (fn) => (onUnauthorized = fn)

const AUTH_PATHS = ['/me', '/login']

async function request(path, { method = 'GET', json, form } = {}) {
  const headers = {}
  let body
  if (json !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(json)
  } else if (form) {
    body = form
  }

  let res
  try {
    res = await fetch('/api' + path, { method, headers, body, credentials: 'same-origin' })
  } catch {
    throw new ApiError(0, "Can't reach the server. Check your connection.")
  }
  if (res.status === 204) return null

  const data = await res.json().catch(() => null)
  if (!res.ok) {
    if (!data && res.status >= 500) {
      throw new ApiError(res.status, "Can't reach the IT-Assist API. It may still be starting, or it has stopped.", { reason: 'unreachable' })
    }
    if (!data && res.status === 404) {
      throw new ApiError(404, 'Something answered on the API address, but it is not the current IT-Assist API (it does not know ' +
        path.split('?')[0] + '). An older backend may still be running.', { reason: 'wrong-api' })
    }
    if (res.status === 401 && !AUTH_PATHS.some((p) => path.startsWith(p))) onUnauthorized()
    throw new ApiError(res.status, data?.error || `Request failed (${res.status})`, data)
  }
  return data
}

const qs = (params) =>
  new URLSearchParams(Object.entries(params).filter(([, v]) => v !== '' && v != null)).toString()

// ---- session ----
export const login = (username, password) => request('/login', { method: 'POST', json: { username, password } })
export const logout = () => request('/logout', { method: 'POST' })
export const getMe = () => request('/me')

// ---- lookups ----
export const getDevices = () => request('/devices')
export const getResolveTypes = () => request('/resolve-types')

// ---- tickets ----
export const listTickets = ({ status = '', q = '', page = 1, pageSize = 10 } = {}) =>
  request('/tickets?' + qs({ status, q, page, pageSize }))
export const countTickets = async (status) => (await listTickets({ status, pageSize: 1 })).total
export const getTicket = (id) => request(`/tickets/${id}`)

export function createTicket({ title, deviceId, description, image }) {
  const form = new FormData()
  form.append('title', title)
  form.append('device_id', String(deviceId))
  form.append('description', description)
  if (image) form.append('image', image)
  return request('/tickets', { method: 'POST', form })
}

export const sendReply = (id, body) => request(`/tickets/${id}/messages`, { method: 'POST', json: { body } })
export const acceptTicket = (id) => request(`/tickets/${id}/accept`, { method: 'POST' })
export const closeTicket = (id, resolveTypeId, note) =>
  request(`/tickets/${id}/close`, { method: 'POST', json: { resolveTypeId, note } })

// ---- notifications ----
export const getNotifications = () => request('/notifications')
export const markNotificationsRead = (ticketId = 0) =>
  request('/notifications/read', { method: 'POST', json: { ticketId } })
