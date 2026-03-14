import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { AuthUser, LoginCredentials, LoginResponse } from '@/lib/api/auth'
import { getCurrentUser, login as loginRequest } from '@/lib/api/auth'
import { clearAuthToken, readAuthToken, writeAuthToken } from '@/lib/auth/token'

export interface AuthStoreUser extends AuthUser {
  avatar: string | null
}

export interface AuthProfileUpdate {
  name: string
  avatar?: string | null
}

export const useAuthStore = defineStore('auth', () => {
  const initialized = ref(false)
  const loading = ref(false)
  const token = ref<string | null>(readAuthToken())
  const user = ref<AuthStoreUser | null>(null)

  const isAuthenticated = computed(() => Boolean(token.value && user.value))

  let initializePromise: Promise<void> | null = null

  function normalizeUser(nextUser: AuthUser | AuthStoreUser): AuthStoreUser {
    return {
      ...nextUser,
      avatar: 'avatar' in nextUser ? (nextUser.avatar ?? null) : null,
    }
  }

  function setUser(nextUser: AuthUser | AuthStoreUser | null) {
    user.value = nextUser ? normalizeUser(nextUser) : null
  }

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
        setUser(response.user)
      } catch {
        setToken(null)
        setUser(null)
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
    setUser(response.user)
    initialized.value = true
    return response
  }

  function updateProfile(profile: AuthProfileUpdate) {
    if (!user.value) {
      return
    }

    user.value = {
      ...user.value,
      name: profile.name,
      avatar: profile.avatar === undefined ? user.value.avatar : profile.avatar,
    }
  }

  function logout() {
    setToken(null)
    setUser(null)
    initialized.value = true
  }

  return { initialized, loading, token, user, isAuthenticated, initialize, login, updateProfile, logout }
})
