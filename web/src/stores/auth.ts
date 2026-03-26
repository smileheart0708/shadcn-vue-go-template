import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { Router } from 'vue-router'
import type { AuthUser, GetCurrentUserOptions, LoginCredentials, LoginResponse } from '@/lib/api/auth'
import type { ChangePasswordInput, UpdateProfileInput } from '@/lib/api/account'
import { getCurrentUser, login as loginRequest, logout as logoutRequest, refreshSession } from '@/lib/api/auth'
import { deleteAccount as deleteAccountRequest, updatePassword as updatePasswordRequest, updateProfile as updateProfileRequest, uploadAvatar as uploadAvatarRequest } from '@/lib/api/account'
import { clearAuthClientHandlers, registerAuthClientHandlers } from '@/lib/api/client'
import { clearAuthToken, readAuthToken, writeAuthToken } from '@/lib/auth/token'

export const useAuthStore = defineStore('auth', () => {
  const initialized = ref(false)
  const initializing = ref(false)
  const refreshing = ref(false)
  const token = ref<string | null>(readAuthToken())
  const user = ref<AuthUser | null>(null)

  const isAuthenticated = computed(() => Boolean(token.value && user.value))

  let initializePromise: Promise<void> | null = null
  let boundRouter: Router | null = null
  let clientHandlersRegistered = false

  function bindRouter(router: Router) {
    boundRouter = router

    if (clientHandlersRegistered) {
      return
    }

    registerAuthClientHandlers({
      refreshAccessToken: async () => {
        refreshing.value = true

        try {
          const response = await refreshSession()
          applySession(response)
          return response.accessToken
        } catch (error) {
          resetSession()
          throw error
        } finally {
          refreshing.value = false
        }
      },
      onUnauthorized: async () => {
        resetSession()
        await redirectToLogin()
      },
    })

    clientHandlersRegistered = true
  }

  function setUser(nextUser: AuthUser | null) {
    if (sameAuthUser(user.value, nextUser)) {
      return
    }

    user.value = nextUser
  }

  function setToken(nextToken: string | null) {
    token.value = nextToken

    if (nextToken) {
      writeAuthToken(nextToken)
      return
    }

    clearAuthToken()
  }

  function applySession(response: LoginResponse) {
    setToken(response.accessToken)
    applyCurrentUser(response.user)
    initialized.value = true
  }

  function resetSession() {
    setToken(null)
    setUser(null)
    initialized.value = true
  }

  async function fetchCurrentUser(options: GetCurrentUserOptions = {}) {
    return getCurrentUser(options)
  }

  function applyCurrentUser(nextUser: AuthUser) {
    setUser(nextUser)
  }

  async function initialize() {
    if (initialized.value && user.value !== null) {
      return
    }

    if (initializePromise) {
      return initializePromise
    }

    initializePromise = (async () => {
      initializing.value = true

      try {
        const currentToken = token.value
        if (currentToken) {
          try {
            const nextUser = await fetchCurrentUser()
            applyCurrentUser(nextUser)
            initialized.value = true
            return
          } catch {
            resetSession()
          }
        }

        try {
          const response = await refreshSession()
          applySession(response)
        } catch {
          resetSession()
        }
      } finally {
        initializing.value = false
        initializePromise = null
      }
    })()

    return initializePromise
  }

  async function login(credentials: LoginCredentials): Promise<LoginResponse> {
    const response = await loginRequest(credentials)
    applySession(response)
    return response
  }

  async function saveProfile(profile: UpdateProfileInput) {
    const nextUser = await updateProfileRequest(profile)
    applyCurrentUser(nextUser)
    return user.value ?? nextUser
  }

  async function uploadAvatar(file: File) {
    const nextUser = await uploadAvatarRequest(file)
    applyCurrentUser(nextUser)
    return user.value ?? nextUser
  }

  async function changePassword(input: ChangePasswordInput) {
    await updatePasswordRequest(input)
    resetSession()
  }

  async function deleteAccount() {
    await deleteAccountRequest()
    resetSession()
  }

  async function logout() {
    try {
      await logoutRequest()
    } finally {
      resetSession()
    }
  }

  async function redirectToLogin() {
    if (!boundRouter) {
      return
    }

    const currentRoute = boundRouter.currentRoute.value
    const redirect = currentRoute.fullPath !== '/login' ? currentRoute.fullPath : undefined

    await boundRouter.push({
      name: 'login',
      query: redirect ? { redirect } : undefined,
    }).catch(() => undefined)
  }

  return {
    initialized,
    initializing,
    refreshing,
    token,
    user,
    isAuthenticated,
    bindRouter,
    fetchCurrentUser,
    applyCurrentUser,
    initialize,
    login,
    saveProfile,
    uploadAvatar,
    changePassword,
    deleteAccount,
    logout,
    resetSession,
  }
})

function sameAuthUser(currentUser: AuthUser | null, nextUser: AuthUser | null) {
  if (currentUser === nextUser) {
    return true
  }

  if (!currentUser || !nextUser) {
    return currentUser === nextUser
  }

  return (
    currentUser.id === nextUser.id &&
    currentUser.username === nextUser.username &&
    currentUser.email === nextUser.email &&
    currentUser.avatarUrl === nextUser.avatarUrl &&
    currentUser.role === nextUser.role &&
    currentUser.mustChangePassword === nextUser.mustChangePassword
  )
}

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    clearAuthClientHandlers()
  })
}
