import { apiRequest } from '@/lib/api/client'

export interface AuthUser {
  id: string
  email: string
  name: string
}

export interface LoginCredentials {
  email: string
  password: string
}

export interface LoginResponse {
  accessToken: string
  tokenType: string
  expiresAt: string
  user: AuthUser
}

export interface CurrentUserResponse {
  user: AuthUser
}

export function login(credentials: LoginCredentials) {
  return apiRequest<LoginResponse>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify(credentials),
  })
}

export function getCurrentUser() {
  return apiRequest<CurrentUserResponse>('/api/auth/me')
}
