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
  ],
})

export default router
