CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL COLLATE NOCASE UNIQUE,
    email TEXT NULL COLLATE NOCASE,
    avatar_path TEXT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    security_version INTEGER NOT NULL DEFAULT 1,
    disabled_at INTEGER NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('owner', 'user'))
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
ON users(email)
WHERE email IS NOT NULL AND email <> '';

CREATE INDEX IF NOT EXISTS users_status_idx
ON users(status);

CREATE UNIQUE INDEX IF NOT EXISTS users_owner_unique_idx
ON users(role)
WHERE role = 'owner';

CREATE TABLE IF NOT EXISTS credentials (
    user_id INTEGER PRIMARY KEY,
    password_hash TEXT NOT NULL,
    password_changed_at INTEGER NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

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

CREATE TABLE IF NOT EXISTS account_policies (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    public_registration_enabled INTEGER NOT NULL CHECK (public_registration_enabled IN (0, 1)),
    self_service_account_deletion_enabled INTEGER NOT NULL CHECK (self_service_account_deletion_enabled IN (0, 1))
);

INSERT OR IGNORE INTO install_state (id, setup_state, owner_user_id, setup_completed_at, created_at, updated_at)
VALUES (1, 'pending', NULL, NULL, strftime('%s', 'now'), strftime('%s', 'now'));

INSERT OR IGNORE INTO account_policies (id, public_registration_enabled, self_service_account_deletion_enabled)
VALUES (1, 0, 0);
