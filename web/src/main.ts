import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import router from './router'

async function enableMocking() {
  if (!import.meta.env.DEV || !import.meta.env.VITE_API_MOCKING) {
    return
  }

  const { worker } = await import('@/mocks/browser')
  await worker.start({ onUnhandledRequest: 'bypass' })
}

async function bootstrap() {
  await enableMocking()

  createApp(App).use(router).mount('#app')
}

void bootstrap()
