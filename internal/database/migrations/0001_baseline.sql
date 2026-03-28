CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL COLLATE NOCASE UNIQUE,
    email TEXT NULL COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    avatar_path TEXT NULL,
    role INTEGER NOT NULL CHECK (role IN (0, 1, 2)),
    bootstrap_password_active INTEGER NOT NULL DEFAULT 0,
    auth_version INTEGER NOT NULL DEFAULT 1,
    password_changed_at INTEGER NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    is_banned INTEGER NOT NULL DEFAULT 0,
    banned_at INTEGER NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
ON users(email)
WHERE email IS NOT NULL AND email <> '';

CREATE TABLE IF NOT EXISTS auth_refresh_sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    token_hash TEXT NOT NULL,
    issued_at INTEGER NOT NULL,
    last_used_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    idle_expires_at INTEGER NOT NULL,
    revoked_at INTEGER NULL,
    revoke_reason TEXT NULL,
    ip TEXT NULL,
    user_agent TEXT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS auth_refresh_sessions_user_idx
ON auth_refresh_sessions(user_id);

CREATE INDEX IF NOT EXISTS auth_refresh_sessions_revoked_idx
ON auth_refresh_sessions(revoked_at);

CREATE TABLE IF NOT EXISTS auth_login_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NULL,
    session_id TEXT NULL,
    identifier TEXT NULL,
    ip TEXT NULL,
    user_agent TEXT NULL,
    event_type TEXT NOT NULL,
    success INTEGER NOT NULL,
    failure_reason TEXT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (session_id) REFERENCES auth_refresh_sessions(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS auth_login_logs_created_at_idx
ON auth_login_logs(created_at DESC);

CREATE INDEX IF NOT EXISTS auth_login_logs_user_idx
ON auth_login_logs(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS system_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    registration_mode TEXT NOT NULL CHECK (registration_mode IN ('disabled', 'password')),
    updated_at INTEGER NOT NULL
);

INSERT OR IGNORE INTO system_settings (id, registration_mode, updated_at)
VALUES (1, 'password', strftime('%s', 'now'));
