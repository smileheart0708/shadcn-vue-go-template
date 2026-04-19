CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL COLLATE NOCASE UNIQUE,
    email TEXT NULL COLLATE NOCASE,
    avatar_path TEXT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    security_version INTEGER NOT NULL DEFAULT 1,
    disabled_at INTEGER NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
ON users(email)
WHERE email IS NOT NULL AND email <> '';

CREATE INDEX IF NOT EXISTS users_status_idx
ON users(status);

CREATE TABLE IF NOT EXISTS credentials (
    user_id INTEGER PRIMARY KEY,
    password_hash TEXT NOT NULL,
    password_changed_at INTEGER NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS roles (
    key TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS permissions (
    key TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_key TEXT NOT NULL,
    permission_key TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (role_key, permission_key),
    FOREIGN KEY (role_key) REFERENCES roles(key) ON DELETE CASCADE,
    FOREIGN KEY (permission_key) REFERENCES permissions(key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id INTEGER PRIMARY KEY,
    role_key TEXT NOT NULL,
    assigned_at INTEGER NOT NULL,
    assigned_by_user_id INTEGER NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_key) REFERENCES roles(key) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX IF NOT EXISTS user_roles_owner_unique_idx
ON user_roles(role_key)
WHERE role_key = 'owner';

CREATE INDEX IF NOT EXISTS user_roles_role_idx
ON user_roles(role_key);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    refresh_token_hash TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    last_used_at INTEGER NOT NULL,
    last_rotated_at INTEGER NULL,
    expires_at INTEGER NOT NULL,
    idle_expires_at INTEGER NOT NULL,
    revoked_at INTEGER NULL,
    revoke_reason TEXT NULL,
    ip TEXT NULL,
    user_agent TEXT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS auth_sessions_user_idx
ON auth_sessions(user_id);

CREATE INDEX IF NOT EXISTS auth_sessions_revoked_idx
ON auth_sessions(revoked_at);

CREATE INDEX IF NOT EXISTS auth_sessions_active_user_idx
ON auth_sessions(user_id)
WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_user_id INTEGER NULL,
    subject_user_id INTEGER NULL,
    auth_session_id TEXT NULL,
    event_type TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('success', 'failure')),
    reason TEXT NULL,
    ip TEXT NULL,
    user_agent TEXT NULL,
    metadata_json TEXT NULL,
    occurred_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS audit_logs_occurred_at_idx
ON audit_logs(occurred_at DESC);

CREATE INDEX IF NOT EXISTS audit_logs_actor_idx
ON audit_logs(actor_user_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS audit_logs_subject_idx
ON audit_logs(subject_user_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS audit_logs_event_idx
ON audit_logs(event_type, occurred_at DESC);

CREATE TABLE IF NOT EXISTS install_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    setup_state TEXT NOT NULL CHECK (setup_state IN ('pending', 'completed')),
    owner_user_id INTEGER NULL,
    setup_completed_at INTEGER NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS system_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    auth_mode TEXT NOT NULL CHECK (auth_mode IN ('single_user', 'multi_user')),
    registration_mode TEXT NOT NULL CHECK (registration_mode IN ('disabled', 'public')),
    password_login_enabled INTEGER NOT NULL CHECK (password_login_enabled IN (0, 1)),
    admin_user_create_enabled INTEGER NOT NULL CHECK (admin_user_create_enabled IN (0, 1)),
    self_service_account_deletion_enabled INTEGER NOT NULL CHECK (self_service_account_deletion_enabled IN (0, 1)),
    updated_at INTEGER NOT NULL
);

INSERT OR IGNORE INTO install_state (id, setup_state, owner_user_id, setup_completed_at, created_at, updated_at)
VALUES (1, 'pending', NULL, NULL, strftime('%s', 'now'), strftime('%s', 'now'));

INSERT OR IGNORE INTO system_settings (
    id,
    auth_mode,
    registration_mode,
    password_login_enabled,
    admin_user_create_enabled,
    self_service_account_deletion_enabled,
    updated_at
)
VALUES (
    1,
    'single_user',
    'disabled',
    1,
    1,
    1,
    strftime('%s', 'now')
);

INSERT OR IGNORE INTO roles (key, created_at)
VALUES
    ('owner', strftime('%s', 'now')),
    ('admin', strftime('%s', 'now')),
    ('user', strftime('%s', 'now'));

INSERT OR IGNORE INTO permissions (key, created_at)
VALUES
    ('system.settings.read', strftime('%s', 'now')),
    ('system.settings.update', strftime('%s', 'now')),
    ('management.users.read', strftime('%s', 'now')),
    ('management.users.create', strftime('%s', 'now')),
    ('management.users.update', strftime('%s', 'now')),
    ('management.users.disable', strftime('%s', 'now')),
    ('management.users.enable', strftime('%s', 'now')),
    ('management.audit_logs.read', strftime('%s', 'now')),
    ('management.system_logs.read', strftime('%s', 'now'));

INSERT OR IGNORE INTO role_permissions (role_key, permission_key, created_at)
VALUES
    ('owner', 'system.settings.read', strftime('%s', 'now')),
    ('owner', 'system.settings.update', strftime('%s', 'now')),
    ('owner', 'management.users.read', strftime('%s', 'now')),
    ('owner', 'management.users.create', strftime('%s', 'now')),
    ('owner', 'management.users.update', strftime('%s', 'now')),
    ('owner', 'management.users.disable', strftime('%s', 'now')),
    ('owner', 'management.users.enable', strftime('%s', 'now')),
    ('owner', 'management.audit_logs.read', strftime('%s', 'now')),
    ('owner', 'management.system_logs.read', strftime('%s', 'now')),
    ('admin', 'management.users.read', strftime('%s', 'now')),
    ('admin', 'management.users.create', strftime('%s', 'now')),
    ('admin', 'management.users.update', strftime('%s', 'now')),
    ('admin', 'management.users.disable', strftime('%s', 'now')),
    ('admin', 'management.users.enable', strftime('%s', 'now'));
