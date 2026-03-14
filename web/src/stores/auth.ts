import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { AuthUser, LoginCredentials, LoginResponse } from '@/lib/api/auth'
import type { ChangePasswordInput, UpdateProfileInput } from '@/lib/api/account'
import { getCurrentUser, login as loginRequest } from '@/lib/api/auth'
import { deleteAccount as deleteAccountRequest, updatePassword as updatePasswordRequest, updateProfile as updateProfileRequest, uploadAvatar as uploadAvatarRequest } from '@/lib/api/account'
import { clearAuthToken, readAuthToken, writeAuthToken } from '@/lib/auth/token'

export const useAuthStore = defineStore('auth', () => {
  const initialized = ref(false)
  const loading = ref(false)
  const token = ref<string | null>(readAuthToken())
  const user = ref<AuthUser | null>(null)

  const isAuthenticated = computed(() => Boolean(token.value && user.value))

  let initializePromise: Promise<void> | null = null

  function setUser(nextUser: AuthUser | null) {
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
        const nextUser = await getCurrentUser()
        setUser(nextUser)
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

  async function saveProfile(profile: UpdateProfileInput) {
    const nextUser = await updateProfileRequest(profile)
    setUser(nextUser)
    return nextUser
  }

  async function uploadAvatar(file: File) {
    const nextUser = await uploadAvatarRequest(file)
    setUser(nextUser)
    return nextUser
  }

  async function changePassword(input: ChangePasswordInput) {
    const nextUser = await updatePasswordRequest(input)
    setUser(nextUser)
    return nextUser
  }

  async function deleteAccount() {
    await deleteAccountRequest()
    logout()
  }

  function logout() {
    setToken(null)
    setUser(null)
    initialized.value = true
  }

  return {
    initialized,
    loading,
    token,
    user,
    isAuthenticated,
    initialize,
    login,
    saveProfile,
    uploadAvatar,
    changePassword,
    deleteAccount,
    logout,
  }
})
