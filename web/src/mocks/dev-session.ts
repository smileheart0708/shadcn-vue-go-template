import { writeAuthToken } from '@/lib/auth/token'
import { ensureMockOwnerSession } from '@/mocks/handlers/auth'

export function initializeMockDevSession() {
  writeAuthToken(ensureMockOwnerSession())
}
