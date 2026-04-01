import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: () => import('../views/DashboardView.vue'),
    },
    {
      path: '/git',
      name: 'git',
      component: () => import('../views/GitView.vue'),
    },
    {
      path: '/commands',
      name: 'commands',
      component: () => import('../views/CommandsView.vue'),
    },
    {
      path: '/docker',
      name: 'docker',
      component: () => import('../views/DockerView.vue'),
    },
    {
      path: '/console',
      name: 'console',
      component: () => import('../views/ConsoleView.vue'),
    },
    {
      path: '/readme',
      name: 'readme',
      component: () => import('../views/ReadmeView.vue'),
    },
    {
      path: '/notes',
      name: 'notes',
      component: () => import('../views/NotesView.vue'),
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('../views/SettingsView.vue'),
    },
  ],
})

export default router
