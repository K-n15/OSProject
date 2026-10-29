import { createRouter, createWebHashHistory } from 'vue-router'
import EmployeeTickets from './views/EmployeeTickets.vue'
import CommonSolutions from './views/CommonSolutions.vue'
import Faq from './views/Faq.vue'
import SupportQueue from './views/SupportQueue.vue'
import Login from './views/auth/Login.vue'
import { setUnauthorizedHandler } from './api'
import { session, checkSession, roleArea, homePath, clearUser } from './session'

const Empty = { render: () => null }

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'root', component: Empty },
    { path: '/login', component: Login, meta: { public: true, title: 'Login' } },

    { path: '/employee/tickets', component: EmployeeTickets, meta: { role: 'employee', title: 'Requested tickets' } },
    { path: '/employee/solutions', component: CommonSolutions, meta: { role: 'employee', title: 'Common solutions' } },
    { path: '/employee/faq', component: Faq, meta: { role: 'employee', title: 'FAQ' } },
    { path: '/support/pending', component: SupportQueue, props: { status: 'pending' }, meta: { role: 'support', title: 'Unanswered tickets' } },
    { path: '/support/ongoing', component: SupportQueue, props: { status: 'ongoing' }, meta: { role: 'support', title: 'Ongoing tickets' } },
    { path: '/support/closed', component: SupportQueue, props: { status: 'closed' }, meta: { role: 'support', title: 'Closed tickets' } },
    { path: '/:pathMatch(.*)*', component: Empty },
  ],
})

router.beforeEach(async (to) => {
  if (!session.checked) await checkSession()
  if (session.error) return true

  if (!session.user) {
    if (to.meta.public) return true
    if (to.meta.role) return { path: '/login', query: { next: to.fullPath } }
    return '/login'
  }
  if (to.meta.public || !to.meta.role || to.meta.role !== roleArea(session.user)) return homePath(session.user)
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · IT-Assist` : 'IT-Assist'
})

setUnauthorizedHandler(() => {
  if (!session.user) return
  clearUser()
  router.replace({ path: '/login', query: { msg: 'expired' } })
})

export default router
