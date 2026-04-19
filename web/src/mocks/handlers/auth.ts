/* eslint-disable @typescript-eslint/array-type, @typescript-eslint/consistent-type-assertions, @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-return, @typescript-eslint/prefer-optional-chain, @typescript-eslint/restrict-template-expressions, @typescript-eslint/strict-boolean-expressions */
import { http, HttpResponse } from 'msw'

type RoleKey = 'owner' | 'admin' | 'user'
type UserStatus = 'active' | 'disabled'
type Capability =
  | 'system.settings.read'
  | 'system.settings.update'
  | 'management.users.read'
  | 'management.users.create'
  | 'management.users.update'
  | 'management.users.disable'
  | 'management.users.enable'
  | 'management.audit_logs.read'
  | 'management.system_logs.read'
  | 'account.delete_self'

interface MockUser {
  id: number
  username: string
  email: string | null
  avatarUrl: string | null
  roleKeys: RoleKey[]
  status: UserStatus
  password: string
  createdAt: string
  updatedAt: string
}

interface MockSession {
  userId: number
  accessToken: string
  expiresAt: string
}

interface MockAuditEntry {
  id: number
  actorUserID: number | null
  subjectUserID: number | null
  authSessionID: string | null
  eventType: string
  outcome: 'success' | 'failure'
  reason: string | null
  ip: string | null
  userAgent: string | null
  metadata?: Record<string, unknown>
  occurredAt: string
}

interface MockState {
  installState: {
    setupState: 'pending' | 'completed'
    setupCompleted: boolean
    ownerUserId: number | null
    completedAt: string | null
  }
  systemSettings: {
    authMode: 'single_user' | 'multi_user'
    registrationMode: 'disabled' | 'public'
    passwordLoginEnabled: boolean
    adminUserCreateEnabled: boolean
    selfServiceAccountDeletionEnabled: boolean
    updatedAt: string
  }
  users: MockUser[]
  session: MockSession | null
  auditLogs: MockAuditEntry[]
  nextUserId: number
  nextAuditId: number
  nextLogId: number
}

function nowISO() {
  return new Date().toISOString()
}

function createInitialState(): MockState {
  return {
    installState: {
      setupState: 'pending',
      setupCompleted: false,
      ownerUserId: null,
      completedAt: null,
    },
    systemSettings: {
      authMode: 'single_user',
      registrationMode: 'disabled',
      passwordLoginEnabled: true,
      adminUserCreateEnabled: true,
      selfServiceAccountDeletionEnabled: true,
      updatedAt: nowISO(),
    },
    users: [],
    session: null,
    auditLogs: [],
    nextUserId: 1,
    nextAuditId: 1,
    nextLogId: 4,
  }
}

const state = createInitialState()

const baseSystemLogs = [
  createSystemLogEntry(1, 'INFO', 'mock', 'Mock system log stream connected.'),
  createSystemLogEntry(2, 'INFO', 'auth', 'No active mock session.'),
  createSystemLogEntry(3, 'DEBUG', 'setup', 'Mock install state is pending.'),
]

function createSystemLogEntry(id: number, level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR', source: string, text: string) {
  return {
    id,
    timestamp: Date.now(),
    level,
    source,
    message: text,
    text,
  }
}

function jsonSuccess(data: unknown, init?: ResponseInit) {
  return HttpResponse.json({ success: true, data }, init)
}

function jsonError(status: number, code: string, message: string) {
  return HttpResponse.json({ success: false, error: { code, message } }, { status })
}

function getPublicAuthConfig() {
  return {
    authMode: state.systemSettings.authMode,
    registrationMode: state.systemSettings.registrationMode,
    passwordLoginEnabled: state.systemSettings.passwordLoginEnabled,
    registrationEnabled: state.systemSettings.authMode === 'multi_user' && state.systemSettings.registrationMode === 'public' && state.systemSettings.passwordLoginEnabled,
  }
}

function getCurrentUserFromRequest(request: Request) {
  if (!state.installState.setupCompleted || state.session === null) {
    return null
  }

  const authorization = request.headers.get('Authorization')
  if (authorization !== `Bearer ${state.session.accessToken}`) {
    return null
  }

  const user = state.users.find((entry) => entry.id === state.session?.userId) ?? null
  if (user === null || user.status !== 'active') {
    return null
  }

  return user
}

function getCurrentUserFromSession() {
  if (state.session === null) {
    return null
  }

  const user = state.users.find((entry) => entry.id === state.session?.userId) ?? null
  if (user === null || user.status !== 'active') {
    return null
  }

  return user
}

function roleOf(user: MockUser) {
  return user.roleKeys[0] ?? 'user'
}

function capabilitiesFor(user: MockUser): Capability[] {
  const capabilities = new Set<Capability>()
  const primaryRole = roleOf(user)

  if (primaryRole === 'owner') {
    capabilities.add('system.settings.read')
    capabilities.add('system.settings.update')
    capabilities.add('management.users.read')
    capabilities.add('management.users.create')
    capabilities.add('management.users.update')
    capabilities.add('management.users.disable')
    capabilities.add('management.users.enable')
    capabilities.add('management.audit_logs.read')
    capabilities.add('management.system_logs.read')
  }

  if (primaryRole === 'admin') {
    capabilities.add('management.users.read')
    capabilities.add('management.users.update')
    capabilities.add('management.users.disable')
    capabilities.add('management.users.enable')
    if (state.systemSettings.authMode === 'multi_user' && state.systemSettings.adminUserCreateEnabled) {
      capabilities.add('management.users.create')
    }
  }

  if (state.systemSettings.selfServiceAccountDeletionEnabled && primaryRole !== 'owner') {
    capabilities.add('account.delete_self')
  }

  if (state.systemSettings.authMode !== 'multi_user') {
    capabilities.delete('management.users.create')
  }

  return [...capabilities].sort()
}

function toViewer(user: MockUser) {
  return {
    identity: {
      id: user.id,
      username: user.username,
      email: user.email,
      avatarUrl: user.avatarUrl,
      status: user.status,
    },
    authorization: {
      roleKeys: [...user.roleKeys],
      capabilities: capabilitiesFor(user),
    },
  }
}

function issueSession(user: MockUser) {
  const accessToken = `mock-access-${user.id}-${Date.now()}`
  const expiresAt = new Date(Date.now() + 10 * 60_000).toISOString()
  state.session = {
    userId: user.id,
    accessToken,
    expiresAt,
  }

  return {
    accessToken,
    tokenType: 'Bearer',
    expiresAt,
    viewer: toViewer(user),
  }
}

function clearSession() {
  state.session = null
}

function appendAudit(eventType: string, outcome: 'success' | 'failure', options: Partial<MockAuditEntry> = {}) {
  const entry: MockAuditEntry = {
    id: state.nextAuditId++,
    actorUserID: options.actorUserID ?? null,
    subjectUserID: options.subjectUserID ?? null,
    authSessionID: options.authSessionID ?? null,
    eventType,
    outcome,
    reason: options.reason ?? null,
    ip: options.ip ?? null,
    userAgent: options.userAgent ?? null,
    metadata: options.metadata,
    occurredAt: nowISO(),
  }

  state.auditLogs = [entry, ...state.auditLogs]
}

function usernameTaken(username: string, excludeUserID?: number) {
  return state.users.some((user) => user.username.toLowerCase() === username.toLowerCase() && user.id !== excludeUserID)
}

function emailTaken(email: string | null, excludeUserID?: number) {
  if (!email) {
    return false
  }

  return state.users.some((user) => user.email?.toLowerCase() === email.toLowerCase() && user.id !== excludeUserID)
}

function requireSetupCompleted() {
  if (!state.installState.setupCompleted) {
    return jsonError(403, 'setup_required', 'Complete setup first.')
  }

  return null
}

function requireAuthenticated(request: Request) {
  const setupResponse = requireSetupCompleted()
  if (setupResponse) {
    return { error: setupResponse, user: null as MockUser | null }
  }

  const user = getCurrentUserFromRequest(request)
  if (user === null) {
    return { error: jsonError(401, 'unauthorized', 'Authentication required.'), user: null as MockUser | null }
  }

  return { error: null, user }
}

function hasCapability(user: MockUser, capability: Capability) {
  return capabilitiesFor(user).includes(capability)
}

function canManageUser(actor: MockUser, target: MockUser, _action: 'update' | 'disable' | 'enable') {
  if (actor.id === target.id) {
    return false
  }

  const actorRole = roleOf(actor)
  const targetRole = roleOf(target)

  if (actorRole === 'owner') {
    if (targetRole === 'owner') {
      return false
    }
    return true
  }

  if (actorRole === 'admin') {
    return targetRole === 'user'
  }

  return false
}

function managedUserActions(actor: MockUser, target: MockUser) {
  const actions: Array<'update' | 'disable' | 'enable'> = []

  if (canManageUser(actor, target, 'update')) {
    actions.push('update')
  }

  if (target.status === 'active' && canManageUser(actor, target, 'disable')) {
    actions.push('disable')
  }

  if (target.status === 'disabled' && canManageUser(actor, target, 'enable')) {
    actions.push('enable')
  }

  return actions
}

function allowedManagedRoles(actor: MockUser) {
  if (!hasCapability(actor, 'management.users.create')) {
    return [] as RoleKey[]
  }

  if (roleOf(actor) === 'owner') {
    return ['user', 'admin'] as RoleKey[]
  }

  if (roleOf(actor) === 'admin') {
    return ['user'] as RoleKey[]
  }

  return [] as RoleKey[]
}

function toManagedUser(user: MockUser, actor: MockUser) {
  return {
    id: user.id,
    username: user.username,
    email: user.email,
    avatarUrl: user.avatarUrl,
    roleKeys: [...user.roleKeys],
    status: user.status,
    createdAt: user.createdAt,
    updatedAt: user.updatedAt,
    actions: managedUserActions(actor, user),
  }
}

function readPositiveInt(url: URL, key: string, fallback: number) {
  const raw = url.searchParams.get(key)
  const parsed = raw ? Number.parseInt(raw, 10) : Number.NaN
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

function buildSystemLogStreamResponse() {
  const latestEntry = createSystemLogEntry(
    state.nextLogId++,
    state.installState.setupCompleted ? 'INFO' : 'WARN',
    state.installState.setupCompleted ? 'auth' : 'setup',
    state.installState.setupCompleted ? 'Mock stream replayed for authenticated session.' : 'Mock stream replayed while setup is pending.',
  )

  const payload = [...baseSystemLogs, latestEntry].map((entry) => `event: log\ndata: ${JSON.stringify(entry)}\n\n`).join('')

  return new HttpResponse(payload, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      Connection: 'keep-alive',
    },
  })
}

async function readJSON(request: Request) {
  return request.json().catch(() => null)
}

export const authHandlers = [
  http.get('/api/install/state', () => jsonSuccess(state.installState)),

  http.post('/api/install/setup', async ({ request }) => {
    if (state.installState.setupCompleted) {
      return jsonError(409, 'setup_completed', 'Setup has already been completed.')
    }

    const payload = await readJSON(request)
    if (!isRecord(payload) || typeof payload.username !== 'string' || typeof payload.password !== 'string') {
      return jsonError(400, 'invalid_request', 'Invalid setup payload.')
    }

    const username = payload.username.trim()
    const email = typeof payload.email === 'string' ? payload.email.trim() || null : null
    if (username === '') {
      return jsonError(400, 'username_required', 'Username is required.')
    }
    if (payload.password.trim().length < 8) {
      return jsonError(400, 'password_too_short', 'Password must be at least 8 characters.')
    }
    if (usernameTaken(username)) {
      return jsonError(409, 'username_taken', 'Username is already in use.')
    }
    if (emailTaken(email)) {
      return jsonError(409, 'email_taken', 'Email is already in use.')
    }

    const createdAt = nowISO()
    const owner: MockUser = {
      id: state.nextUserId++,
      username,
      email,
      avatarUrl: null,
      roleKeys: ['owner'],
      status: 'active',
      password: payload.password,
      createdAt,
      updatedAt: createdAt,
    }

    state.users = [owner]
    state.installState = {
      setupState: 'completed',
      setupCompleted: true,
      ownerUserId: owner.id,
      completedAt: createdAt,
    }
    state.systemSettings = {
      ...state.systemSettings,
      updatedAt: createdAt,
    }

    const sessionResponse = issueSession(owner)
    appendAudit('setup_completed', 'success', {
      actorUserID: owner.id,
      subjectUserID: owner.id,
      authSessionID: `setup-${owner.id}`,
    })

    return jsonSuccess(sessionResponse, { status: 201 })
  }),

  http.get('/api/auth/public-config', () => jsonSuccess(getPublicAuthConfig())),

  http.post('/api/auth/login', async ({ request }) => {
    const setupResponse = requireSetupCompleted()
    if (setupResponse) {
      return setupResponse
    }

    const payload = await readJSON(request)
    if (!isRecord(payload) || typeof payload.identifier !== 'string' || typeof payload.password !== 'string') {
      return jsonError(400, 'invalid_request', 'Invalid login payload.')
    }

    const user = state.users.find((entry) => entry.username === payload.identifier || entry.email === payload.identifier) ?? null
    if (user === null || user.password !== payload.password) {
      appendAudit('login_failed', 'failure', { reason: 'invalid_credentials' })
      return jsonError(401, 'invalid_credentials', 'Invalid credentials.')
    }
    if (user.status !== 'active') {
      appendAudit('login_failed', 'failure', { subjectUserID: user.id, reason: 'account_disabled' })
      return jsonError(403, 'account_disabled', 'Account is disabled.')
    }

    const sessionResponse = issueSession(user)
    appendAudit('login_succeeded', 'success', { actorUserID: user.id, subjectUserID: user.id, authSessionID: `session-${user.id}` })
    return jsonSuccess(sessionResponse)
  }),

  http.post('/api/auth/register', async ({ request }) => {
    const setupResponse = requireSetupCompleted()
    if (setupResponse) {
      return setupResponse
    }

    if (!getPublicAuthConfig().registrationEnabled) {
      return jsonError(403, 'registration_disabled', 'Registration is disabled.')
    }

    const payload = await readJSON(request)
    if (!isRecord(payload) || typeof payload.username !== 'string' || typeof payload.password !== 'string') {
      return jsonError(400, 'invalid_request', 'Invalid registration payload.')
    }

    const username = payload.username.trim()
    const email = typeof payload.email === 'string' ? payload.email.trim() || null : null
    if (username === '') {
      return jsonError(400, 'username_required', 'Username is required.')
    }
    if (payload.password.trim().length < 8) {
      return jsonError(400, 'password_too_short', 'Password must be at least 8 characters.')
    }
    if (usernameTaken(username)) {
      return jsonError(409, 'username_taken', 'Username is already in use.')
    }
    if (emailTaken(email)) {
      return jsonError(409, 'email_taken', 'Email is already in use.')
    }

    const createdAt = nowISO()
    const user: MockUser = {
      id: state.nextUserId++,
      username,
      email,
      avatarUrl: null,
      roleKeys: ['user'],
      status: 'active',
      password: payload.password,
      createdAt,
      updatedAt: createdAt,
    }

    state.users = [...state.users, user]
    const sessionResponse = issueSession(user)
    appendAudit('registration_succeeded', 'success', { actorUserID: user.id, subjectUserID: user.id, authSessionID: `session-${user.id}` })
    return jsonSuccess(sessionResponse, { status: 201 })
  }),

  http.post('/api/auth/refresh', () => {
    const user = getCurrentUserFromSession()
    if (user === null) {
      clearSession()
      appendAudit('refresh_failed', 'failure', { reason: 'invalid_refresh_token' })
      return jsonError(401, 'invalid_refresh_token', 'Refresh token is invalid.')
    }

    const sessionResponse = issueSession(user)
    appendAudit('refresh_succeeded', 'success', { actorUserID: user.id, subjectUserID: user.id, authSessionID: `session-${user.id}` })
    return jsonSuccess(sessionResponse)
  }),

  http.post('/api/auth/logout', () => {
    const user = getCurrentUserFromSession()
    if (user !== null) {
      appendAudit('logout_succeeded', 'success', { actorUserID: user.id, subjectUserID: user.id, authSessionID: `session-${user.id}` })
    }

    clearSession()
    return jsonSuccess({ loggedOut: true })
  }),

  http.get('/api/auth/me', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    return jsonSuccess(toViewer(user))
  }),

  http.patch('/api/account/profile', async ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    const payload = await readJSON(request)
    if (!isRecord(payload) || typeof payload.username !== 'string') {
      return jsonError(400, 'invalid_request', 'Invalid profile payload.')
    }

    const username = payload.username.trim()
    const email = typeof payload.email === 'string' ? payload.email.trim() || null : null
    if (username === '') {
      return jsonError(400, 'username_required', 'Username is required.')
    }
    if (usernameTaken(username, user.id)) {
      return jsonError(409, 'username_taken', 'Username is already in use.')
    }
    if (emailTaken(email, user.id)) {
      return jsonError(409, 'email_taken', 'Email is already in use.')
    }

    user.username = username
    user.email = email
    user.updatedAt = nowISO()
    return jsonSuccess(toViewer(user))
  }),

  http.post('/api/account/avatar', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    user.avatarUrl = `/mock/avatar-${user.id}.png?v=${Date.now()}`
    user.updatedAt = nowISO()
    return jsonSuccess(toViewer(user))
  }),

  http.post('/api/account/password', async ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    const payload = await readJSON(request)
    if (!isRecord(payload) || typeof payload.currentPassword !== 'string' || typeof payload.newPassword !== 'string') {
      return jsonError(400, 'invalid_request', 'Invalid password payload.')
    }
    if (payload.currentPassword !== user.password) {
      return jsonError(400, 'current_password_invalid', 'Current password is invalid.')
    }
    if (payload.newPassword.trim().length < 8) {
      return jsonError(400, 'password_too_short', 'Password must be at least 8 characters.')
    }

    user.password = payload.newPassword
    user.updatedAt = nowISO()
    appendAudit('password_changed', 'success', { actorUserID: user.id, subjectUserID: user.id, authSessionID: `session-${user.id}` })
    clearSession()
    return jsonSuccess({ passwordChanged: true })
  }),

  http.delete('/api/account', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, 'account.delete_self')) {
      return jsonError(403, 'account_delete_forbidden', 'This account cannot delete itself.')
    }

    state.users = state.users.filter((entry) => entry.id !== user.id)
    appendAudit('account_deleted', 'success', { actorUserID: user.id, subjectUserID: user.id, authSessionID: `session-${user.id}`, reason: 'self_service' })
    clearSession()
    return jsonSuccess({ deleted: true })
  }),

  http.get('/api/system/settings', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, 'system.settings.read')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    return jsonSuccess(state.systemSettings)
  }),

  http.patch('/api/system/settings', async ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, 'system.settings.update')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const payload = await readJSON(request)
    if (!isRecord(payload)) {
      return jsonError(400, 'invalid_request', 'Invalid settings payload.')
    }

    if (payload.authMode === 'single_user' || payload.authMode === 'multi_user') {
      state.systemSettings.authMode = payload.authMode
    }
    if (payload.registrationMode === 'disabled' || payload.registrationMode === 'public') {
      state.systemSettings.registrationMode = payload.registrationMode
    }
    if (typeof payload.adminUserCreateEnabled === 'boolean') {
      state.systemSettings.adminUserCreateEnabled = payload.adminUserCreateEnabled
    }
    if (typeof payload.selfServiceAccountDeletionEnabled === 'boolean') {
      state.systemSettings.selfServiceAccountDeletionEnabled = payload.selfServiceAccountDeletionEnabled
    }
    if (state.systemSettings.authMode === 'single_user') {
      state.systemSettings.registrationMode = 'disabled'
    }
    state.systemSettings.updatedAt = nowISO()

    return jsonSuccess(state.systemSettings)
  }),

  http.get('/api/management/users', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, 'management.users.read')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const url = new URL(request.url)
    const query = (url.searchParams.get('q') ?? '').trim().toLowerCase()
    const role = url.searchParams.get('role')
    const status = url.searchParams.get('status')
    const page = readPositiveInt(url, 'page', 1)
    const pageSize = readPositiveInt(url, 'pageSize', 20)

    const filtered = state.users.filter((entry) => {
      if (query !== '' && ![entry.username, entry.email ?? ''].some((value) => value.toLowerCase().includes(query))) {
        return false
      }
      if (role && role !== roleOf(entry)) {
        return false
      }
      if (status && status !== entry.status) {
        return false
      }
      return true
    })

    const start = (page - 1) * pageSize
    const items = filtered.slice(start, start + pageSize).map((entry) => toManagedUser(entry, user))

    return jsonSuccess({
      items,
      page,
      pageSize,
      total: filtered.length,
    })
  }),

  http.post('/api/management/users', async ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, 'management.users.create')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const payload = await readJSON(request)
    if (!isRecord(payload) || typeof payload.username !== 'string' || typeof payload.password !== 'string' || !Array.isArray(payload.roleKeys)) {
      return jsonError(400, 'invalid_request', 'Invalid user payload.')
    }

    const username = payload.username.trim()
    const email = typeof payload.email === 'string' ? payload.email.trim() || null : null
    const roleKeys = payload.roleKeys.filter((value): value is RoleKey => value === 'admin' || value === 'user')
    const allowedRoles = allowedManagedRoles(user)
    if (username === '') {
      return jsonError(400, 'username_required', 'Username is required.')
    }
    if (payload.password.trim().length < 8) {
      return jsonError(400, 'password_too_short', 'Password must be at least 8 characters.')
    }
    if (roleKeys.length !== 1 || !allowedRoles.includes(roleKeys[0])) {
      return jsonError(400, 'invalid_role_keys', 'Invalid role keys.')
    }
    if (usernameTaken(username)) {
      return jsonError(409, 'username_taken', 'Username is already in use.')
    }
    if (emailTaken(email)) {
      return jsonError(409, 'email_taken', 'Email is already in use.')
    }

    const createdAt = nowISO()
    const createdUser: MockUser = {
      id: state.nextUserId++,
      username,
      email,
      avatarUrl: null,
      roleKeys,
      status: 'active',
      password: payload.password,
      createdAt,
      updatedAt: createdAt,
    }

    state.users = [...state.users, createdUser]
    appendAudit('managed_user_created', 'success', { actorUserID: user.id, subjectUserID: createdUser.id })
    return jsonSuccess(toManagedUser(createdUser, user), { status: 201 })
  }),

  http.patch('/api/management/users/:id', async ({ request, params }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    const targetID = Number.parseInt(String(params.id), 10)
    const target = state.users.find((entry) => entry.id === targetID) ?? null
    if (target === null) {
      return jsonError(404, 'user_not_found', 'User not found.')
    }
    if (!canManageUser(user, target, 'update')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const payload = await readJSON(request)
    if (!isRecord(payload) || typeof payload.username !== 'string' || !Array.isArray(payload.roleKeys)) {
      return jsonError(400, 'invalid_request', 'Invalid user payload.')
    }

    const username = payload.username.trim()
    const email = typeof payload.email === 'string' ? payload.email.trim() || null : null
    const roleKeys = payload.roleKeys.filter((value): value is RoleKey => value === 'admin' || value === 'user')
    const allowedRoles = roleOf(user) === 'owner' ? (['user', 'admin'] as RoleKey[]) : (['user'] as RoleKey[])
    if (username === '') {
      return jsonError(400, 'username_required', 'Username is required.')
    }
    if (roleKeys.length !== 1 || !allowedRoles.includes(roleKeys[0])) {
      return jsonError(400, 'invalid_role_keys', 'Invalid role keys.')
    }
    if (usernameTaken(username, target.id)) {
      return jsonError(409, 'username_taken', 'Username is already in use.')
    }
    if (emailTaken(email, target.id)) {
      return jsonError(409, 'email_taken', 'Email is already in use.')
    }

    target.username = username
    target.email = email
    target.roleKeys = roleKeys
    target.updatedAt = nowISO()

    appendAudit('managed_user_updated', 'success', { actorUserID: user.id, subjectUserID: target.id })
    return jsonSuccess(toManagedUser(target, user))
  }),

  http.post('/api/management/users/:id/disable', ({ request, params }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    const targetID = Number.parseInt(String(params.id), 10)
    const target = state.users.find((entry) => entry.id === targetID) ?? null
    if (target === null) {
      return jsonError(404, 'user_not_found', 'User not found.')
    }
    if (!canManageUser(user, target, 'disable')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    target.status = 'disabled'
    target.updatedAt = nowISO()
    if (state.session?.userId === target.id) {
      clearSession()
    }
    appendAudit('managed_user_disabled', 'success', { actorUserID: user.id, subjectUserID: target.id })
    return jsonSuccess(toManagedUser(target, user))
  }),

  http.post('/api/management/users/:id/enable', ({ request, params }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    const targetID = Number.parseInt(String(params.id), 10)
    const target = state.users.find((entry) => entry.id === targetID) ?? null
    if (target === null) {
      return jsonError(404, 'user_not_found', 'User not found.')
    }
    if (!canManageUser(user, target, 'enable')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    target.status = 'active'
    target.updatedAt = nowISO()
    appendAudit('managed_user_enabled', 'success', { actorUserID: user.id, subjectUserID: target.id })
    return jsonSuccess(toManagedUser(target, user))
  }),

  http.get('/api/management/audit-logs', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, 'management.audit_logs.read')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const url = new URL(request.url)
    const page = readPositiveInt(url, 'page', 1)
    const pageSize = readPositiveInt(url, 'pageSize', 50)
    const start = (page - 1) * pageSize

    return jsonSuccess({
      items: state.auditLogs.slice(start, start + pageSize),
      page,
      pageSize,
      total: state.auditLogs.length,
    })
  }),

  http.get('/api/management/system-logs/stream', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, 'management.system_logs.read')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    return buildSystemLogStreamResponse()
  }),
]

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
