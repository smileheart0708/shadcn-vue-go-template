let authToken: string | null = null

export function readAuthToken(): string | null {
  return authToken
}

export function writeAuthToken(token: string): void {
  authToken = token
}

export function clearAuthToken(): void {
  authToken = null
}
