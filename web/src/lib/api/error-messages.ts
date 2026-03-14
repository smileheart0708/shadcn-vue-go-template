import type { APIError } from '@/lib/api/client'

const API_ERROR_MESSAGE_KEYS = {
  invalid_credentials: 'apiError.invalidCredentials',
  unauthorized: 'apiError.unauthorized',
  username_required: 'apiError.usernameRequired',
  username_taken: 'apiError.usernameTaken',
  email_taken: 'apiError.emailTaken',
  current_password_invalid: 'apiError.currentPasswordInvalid',
  password_too_short: 'apiError.passwordTooShort',
  avatar_required: 'apiError.avatarRequired',
  avatar_invalid_type: 'apiError.avatarInvalidType',
  avatar_too_large: 'apiError.avatarTooLarge',
  super_admin_delete_forbidden: 'apiError.superAdminDeleteForbidden',
  profile_update_failed: 'apiError.profileUpdateFailed',
  avatar_upload_failed: 'apiError.avatarUploadFailed',
  avatar_update_failed: 'apiError.avatarUploadFailed',
  password_update_failed: 'apiError.passwordUpdateFailed',
  account_delete_failed: 'apiError.accountDeleteFailed',
} as const

type Translate = (key: string) => string

export function getAPIErrorMessage(t: Translate, error: unknown, fallbackKey = 'apiError.unknown'): string {
  if (isAPIError(error) && error.code) {
    const key = getAPIErrorMessageKey(error.code)
    if (key) {
      return t(key)
    }
  }

  return t(fallbackKey)
}

function getAPIErrorMessageKey(code: string): (typeof API_ERROR_MESSAGE_KEYS)[keyof typeof API_ERROR_MESSAGE_KEYS] | null {
  switch (code) {
    case 'invalid_credentials':
      return API_ERROR_MESSAGE_KEYS.invalid_credentials
    case 'unauthorized':
      return API_ERROR_MESSAGE_KEYS.unauthorized
    case 'username_required':
      return API_ERROR_MESSAGE_KEYS.username_required
    case 'username_taken':
      return API_ERROR_MESSAGE_KEYS.username_taken
    case 'email_taken':
      return API_ERROR_MESSAGE_KEYS.email_taken
    case 'current_password_invalid':
      return API_ERROR_MESSAGE_KEYS.current_password_invalid
    case 'password_too_short':
      return API_ERROR_MESSAGE_KEYS.password_too_short
    case 'avatar_required':
      return API_ERROR_MESSAGE_KEYS.avatar_required
    case 'avatar_invalid_type':
      return API_ERROR_MESSAGE_KEYS.avatar_invalid_type
    case 'avatar_too_large':
      return API_ERROR_MESSAGE_KEYS.avatar_too_large
    case 'profile_update_failed':
      return API_ERROR_MESSAGE_KEYS.profile_update_failed
    case 'avatar_upload_failed':
    case 'avatar_update_failed':
      return API_ERROR_MESSAGE_KEYS.avatar_upload_failed
    case 'password_update_failed':
      return API_ERROR_MESSAGE_KEYS.password_update_failed
    case 'account_delete_failed':
      return API_ERROR_MESSAGE_KEYS.account_delete_failed
    case 'super_admin_delete_forbidden':
      return API_ERROR_MESSAGE_KEYS.super_admin_delete_forbidden
    default:
      return null
  }
}

function isAPIError(error: unknown): error is APIError {
  return error instanceof Error && error.name === 'APIError' && 'status' in error
}
