import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      meta: { order: 0 },
      component: () => import('../views/DashboardView.vue'),
    },
    {
      path: '/git',
      name: 'git',
      meta: { order: 1 },
      component: () => import('../views/GitView.vue'),
    },
    {
      path: '/commands',
      name: 'commands',
      meta: { order: 2 },
      component: () => import('../views/CommandsView.vue'),
    },
    {
      path: '/docker',
      name: 'docker',
      meta: { order: 3 },
      component: () => import('../views/DockerView.vue'),
    },
    {
      path: '/console',
      name: 'console',
      meta: { order: 4 },
      component: () => import('../views/ConsoleView.vue'),
    },
    {
      path: '/readme',
      name: 'readme',
      meta: { order: 5 },
      component: () => import('../views/ReadmeView.vue'),
    },
    {
      path: '/notes',
      name: 'notes',
      meta: { order: 6 },
      component: () => import('../views/NotesView.vue'),
    },
    {
      path: '/settings',
      name: 'settings',
      meta: { order: 8 },
      component: () => import('../views/SettingsView.vue'),
    },
    {
      path: '/editor',
      name: 'editor',
      meta: { order: 7 },
      component: () => import('../views/EditorView.vue'),
    },
    {
      path: '/gitlab',
      name: 'gitlab',
      meta: { order: 9 },
      component: () => import('../views/GitLabView.vue'),
    },
  ],
})

export default router
