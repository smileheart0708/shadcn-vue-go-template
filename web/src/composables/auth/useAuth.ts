import { computed, reactive, readonly } from 'vue'
import type { AuthUser, LoginCredentials, LoginResponse } from '@/lib/api/auth'
import { getCurrentUser, login as loginRequest } from '@/lib/api/auth'
import { clearAuthToken, readAuthToken, writeAuthToken } from '@/lib/auth/token'

interface AuthState {
  initialized: boolean
  loading: boolean
  token: string | null
  user: AuthUser | null
}

const state = reactive<AuthState>({
  initialized: false,
  loading: false,
  token: readAuthToken(),
  user: null,
})

const isAuthenticated = computed(() => Boolean(state.token && state.user))

let initializePromise: Promise<void> | null = null

function storeToken(token: string | null) {
  state.token = token

  if (token) {
    writeAuthToken(token)
    return
  }

  clearAuthToken()
}

export function useAuth() {
  async function initialize() {
    if (state.initialized) {
      return
    }

    if (initializePromise) {
      return initializePromise
    }

    if (!state.token) {
      state.initialized = true
      state.user = null
      return
    }

    initializePromise = (async () => {
      state.loading = true

      try {
        const response = await getCurrentUser()
        state.user = response.user
      }
      catch {
        storeToken(null)
        state.user = null
      }
      finally {
        state.initialized = true
        state.loading = false
        initializePromise = null
      }
    })()

    return initializePromise
  }

  async function login(credentials: LoginCredentials): Promise<LoginResponse> {
    const response = await loginRequest(credentials)
    storeToken(response.accessToken)
    state.user = response.user
    state.initialized = true
    return response
  }

  function logout() {
    storeToken(null)
    state.user = null
    state.initialized = true
  }

  return {
    state: readonly(state),
    isAuthenticated,
    initialize,
    login,
    logout,
  }
}
