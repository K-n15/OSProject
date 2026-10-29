// Stand-in for the Go API, for previewing the UI without Docker or Go.
// In memory only; never deploy it. Same demo accounts as backend/migrations/001_init.sql:
//   ivan / itassist-ivan, ines / itassist-ines (IT)    emma / itassist-emma, noah / itassist-noah (employees)
// Run `npm run mock`, then `npm run dev` in another terminal.
import http from 'node:http'
import crypto from 'node:crypto'
import { Readable } from 'node:stream'

const PORT = 8080
const MAX_ONGOING = 3
const now = () => new Date().toISOString()

const users = [
  { id: 1, username: 'ivan', name: 'Ivan (IT)', role: 'support' },
  { id: 2, username: 'ines', name: 'Ines (IT)', role: 'support' },
  { id: 3, username: 'emma', name: 'Emma Employee', role: 'employee' },
  { id: 4, username: 'noah', name: 'Noah Employee', role: 'employee' },
].map((u) => ({ ...u, password: `itassist-${u.username}`, createdAt: now() }))
const sessions = new Map()
const devices = ['Laptop', 'Desktop', 'Printer', 'Phone or tablet', 'Network or Wi-Fi', 'Other'].map((name, i) => ({ id: i + 1, name }))
const resolveTypes = ['Fixed', 'Workaround provided', 'Guidance given', 'Cannot reproduce', 'Duplicate', 'Escalated to vendor'].map((name, i) => ({ id: i + 1, name }))
const tickets = []
const notifications = []
let nextTicket = 1
let nextMsg = 1
let nextNotif = 1
const notify = (userId, ticketId, kind, actorId) =>
  notifications.push({ id: nextNotif++, userId, ticketId, kind, actorId, createdAt: now(), readAt: null })

class HttpError extends Error { constructor(status, message) { super(message); this.status = status } }
const fail = (status, message) => { throw new HttpError(status, message) }
const publicUser = ({ password, ...u }) => u
const tokenOf = (req) => /(?:^|;\s*)itassist_session=([^;]+)/.exec(req.headers.cookie || '')?.[1]
const sessionUser = (req) => users.find((u) => u.id === sessions.get(tokenOf(req))) || null

function paginate(list, url) {
  const page = Math.max(1, Number(url.searchParams.get('page')) || 1)
  const pageSize = Math.min(50, Math.max(1, Number(url.searchParams.get('pageSize')) || 10))
  return { items: list.slice((page - 1) * pageSize, page * pageSize), total: list.length, page, pageSize }
}

async function readBody(req) {
  if ((req.headers['content-type'] || '').startsWith('multipart/form-data')) {
    const request = new Request('http://local', { method: 'POST', headers: req.headers, body: Readable.toWeb(req), duplex: 'half' })
    return Object.fromEntries(await request.formData())
  }
  let raw = ''
  for await (const chunk of req) raw += chunk
  try { return raw ? JSON.parse(raw) : {} } catch { fail(400, 'invalid JSON body') }
}

const routes = []
const on = (method, pattern, handler, auth = null) =>
  routes.push({ method, re: new RegExp('^' + pattern.replace(/\{id\}/g, '(\\d+)') + '$'), handler, auth })

on('POST', '/api/login', ({ body, res }) => {
  const user = users.find((u) => u.username.toLowerCase() === String(body.username || '').toLowerCase())
  if (!user || user.password !== body.password) fail(401, 'Incorrect username or password')
  const token = crypto.randomBytes(32).toString('base64url')
  sessions.set(token, user.id)
  res.setHeader('Set-Cookie', `itassist_session=${token}; Path=/; HttpOnly; SameSite=Lax`)
  return publicUser(user)
})
on('POST', '/api/logout', ({ req, res }) => {
  sessions.delete(tokenOf(req))
  res.setHeader('Set-Cookie', 'itassist_session=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0')
  return [204]
})
on('GET', '/api/me', ({ user }) => publicUser(user), 'any')
on('GET', '/api/devices', () => devices, 'any')
on('GET', '/api/resolve-types', () => resolveTypes, 'any')

const visible = (t, user) => user.role === 'support' || t.requesterId === user.id
on('GET', '/api/tickets', ({ url, user }) => {
  const status = url.searchParams.get('status') || ''
  const q = (url.searchParams.get('q') || '').toLowerCase()
  let list = tickets.filter((t) => visible(t, user) && (!status || t.status === status) && t.issue.toLowerCase().includes(q))
  if (user.role === 'support' && status === 'ongoing') list = list.filter((t) => t.assigneeId === user.id)
  list.sort(user.role === 'employee' ? (a, b) => b.reported.localeCompare(a.reported)
    : status === 'closed' ? (a, b) => b.closed.localeCompare(a.closed) : (a, b) => a.reported.localeCompare(b.reported))
  return paginate(list.map(({ messages, ...t }) => t), url)
}, 'any')
on('GET', '/api/tickets/{id}', ({ params, user }) => {
  const t = tickets.find((x) => x.id === Number(params[0]))
  if (!t || !visible(t, user)) fail(404, 'not found')
  return t
}, 'any')
on('POST', '/api/tickets', ({ body, user }) => {
  if (user.role !== 'employee') fail(403, 'not available for your role')
  const device = devices.find((d) => d.id === Number(body.device_id))
  if (!body.title || !body.description || !device) fail(400, 'title (max 80), device_id and description are required')
  const t = { id: nextTicket++, issue: body.title, device: device.name, description: body.description, status: 'pending', requesterId: user.id,
    assigneeId: null, resolveType: null, resolveNote: null, reported: now(), closed: null, messages: [], attachments: [] }
  t.updated = t.reported
  tickets.push(t)
  users.filter((u) => u.role === 'support').forEach((u) => notify(u.id, t.id, 'new_ticket', user.id))
  return [201, t]
}, 'any')
on('POST', '/api/tickets/{id}/messages', ({ params, body, user }) => {
  const t = tickets.find((x) => x.id === Number(params[0]))
  if (!t || !visible(t, user)) fail(404, 'not found')
  if (t.status === 'closed') fail(409, 'the ticket is not in a state that allows this')
  if (user.role === 'support' && t.assigneeId !== user.id) fail(403, 'you are not allowed to do that')
  t.messages.push({ id: nextMsg++, authorId: user.id, from: user.role === 'support' ? 'support' : 'employee', text: body.body, at: now() })
  t.updated = now()
  if (user.role === 'support') notify(t.requesterId, t.id, 'reply', user.id)
  else if (t.assigneeId) notify(t.assigneeId, t.id, 'reply', user.id)
  return [201]
}, 'any')
on('POST', '/api/tickets/{id}/accept', ({ params, user }) => {
  const t = tickets.find((x) => x.id === Number(params[0]))
  if (!t) fail(404, 'not found')
  if (tickets.filter((x) => x.assigneeId === user.id && x.status === 'ongoing').length >= MAX_ONGOING)
    fail(409, `you already have ${MAX_ONGOING} ongoing tickets; close one before accepting another`)
  if (t.status !== 'pending') fail(409, 'the ticket is not in a state that allows this')
  Object.assign(t, { status: 'ongoing', assigneeId: user.id, updated: now() })
  notify(t.requesterId, t.id, 'accepted', user.id)
  return [204]
}, 'support')
on('POST', '/api/tickets/{id}/close', ({ params, body, user }) => {
  const t = tickets.find((x) => x.id === Number(params[0]))
  if (!t) fail(404, 'not found')
  if (t.status !== 'ongoing' || t.assigneeId !== user.id) fail(409, 'the ticket is not in a state that allows this')
  const rt = resolveTypes.find((r) => r.id === Number(body.resolveTypeId))
  if (!rt) fail(400, 'resolveTypeId is required')
  Object.assign(t, { status: 'closed', resolveType: rt.name, resolveNote: body.note || null, closed: now(), updated: now() })
  notify(t.requesterId, t.id, 'closed', user.id)
  return [204]
}, 'support')

on('GET', '/api/notifications', ({ user }) => {
  const mine = notifications.filter((n) => n.userId === user.id)
  const items = mine.slice().reverse().slice(0, 20).map((n) => {
    const t = tickets.find((x) => x.id === n.ticketId)
    return { id: n.id, ticketId: n.ticketId, ticketTitle: t.issue, ticketStatus: t.status, kind: n.kind,
      actorName: users.find((u) => u.id === n.actorId)?.name || '', createdAt: n.createdAt, read: !!n.readAt }
  })
  return { items, unread: mine.filter((n) => !n.readAt).length }
}, 'any')
on('POST', '/api/notifications/read', ({ body, user }) => {
  const ticketId = Number(body.ticketId) || 0
  notifications.filter((n) => n.userId === user.id && !n.readAt && (!ticketId || n.ticketId === ticketId)).forEach((n) => (n.readAt = now()))
  return [204]
}, 'any')

http.createServer(async (req, res) => {
  const url = new URL(req.url, 'http://local')
  res.setHeader('Content-Type', 'application/json')
  try {
    const route = routes.find((r) => r.method === req.method && r.re.test(url.pathname))
    if (!route) fail(404, 'not found')
    const user = sessionUser(req)
    if (route.auth && !user) fail(401, 'authentication required')
    if (route.auth === 'support' && user.role !== 'support') fail(403, 'not available for your role')
    const body = req.method === 'POST' ? await readBody(req) : {}
    const out = await route.handler({ req, res, url, body, user, params: route.re.exec(url.pathname).slice(1) })
    const [status, data] = Array.isArray(out) && typeof out[0] === 'number' ? out : [200, out]
    res.statusCode = status
    res.end(data === undefined ? '' : JSON.stringify(data))
  } catch (e) {
    res.statusCode = e.status || 500
    res.end(JSON.stringify({ error: e.status ? e.message : 'internal error' }))
    if (!e.status) console.error(e)
  }
}).listen(PORT, () => console.log(`Mock IT-Assist API on http://localhost:${PORT} (demo accounts as in 001_init.sql)`))
