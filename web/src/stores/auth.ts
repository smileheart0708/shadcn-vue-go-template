import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { AuthUser, LoginCredentials, LoginResponse } from '@/lib/api/auth'
import { getCurrentUser, login as loginRequest } from '@/lib/api/auth'
import { clearAuthToken, readAuthToken, writeAuthToken } from '@/lib/auth/token'

export const useAuthStore = defineStore('auth', () => {
  const initialized = ref(false)
  const loading = ref(false)
  const token = ref<string | null>(readAuthToken())
  const user = ref<AuthUser | null>(null)

  const isAuthenticated = computed(() => Boolean(token.value && user.value))

  let initializePromise: Promise<void> | null = null

  function setToken(nextToken: string | null) {
    token.value = nextToken

    if (nextToken) {
      writeAuthToken(nextToken)
      return
    }

    clearAuthToken()
  }

  async function initialize() {
    if (initialized.value) {
      return
    }

    if (initializePromise) {
      return initializePromise
    }

    if (!token.value) {
      initialized.value = true
      user.value = null
      return
    }

    initializePromise = (async () => {
      loading.value = true

      try {
        const response = await getCurrentUser()
        user.value = response.user
      } catch {
        setToken(null)
        user.value = null
      } finally {
        initialized.value = true
        loading.value = false
        initializePromise = null
      }
    })()

    return initializePromise
  }

  async function login(credentials: LoginCredentials): Promise<LoginResponse> {
    const response = await loginRequest(credentials)
    setToken(response.accessToken)
    user.value = response.user
    initialized.value = true
    return response
  }

  function logout() {
    setToken(null)
    user.value = null
    initialized.value = true
  }

  return { initialized, loading, token, user, isAuthenticated, initialize, login, logout }
})
