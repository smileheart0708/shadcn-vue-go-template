package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrRefreshSessionNotFound = errors.New("users: refresh session not found")

type RefreshSession struct {
	ID            string
	UserID        int64
	TokenHash     string
	IssuedAt      time.Time
	LastUsedAt    time.Time
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
	RevokedAt     *time.Time
	RevokeReason  *string
	IP            *string
	UserAgent     *string
}

type CreateRefreshSessionParams struct {
	ID            string
	UserID        int64
	TokenHash     string
	IssuedAt      time.Time
	LastUsedAt    time.Time
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
	IP            *string
	UserAgent     *string
}

type RotateRefreshSessionParams struct {
	ID            string
	TokenHash     string
	LastUsedAt    time.Time
	IdleExpiresAt time.Time
}

type AuthLogParams struct {
	UserID        *int64
	SessionID     *string
	Identifier    *string
	IP            *string
	UserAgent     *string
	EventType     string
	Success       bool
	FailureReason *string
	CreatedAt     time.Time
}

type AuthLogRecord struct {
	ID            int64
	UserID        *int64
	SessionID     *string
	Identifier    *string
	IP            *string
	UserAgent     *string
	EventType     string
	Success       bool
	FailureReason *string
	CreatedAt     time.Time
}

func (s *Store) CreateRefreshSession(ctx context.Context, params CreateRefreshSessionParams) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO auth_refresh_sessions (
			id,
			user_id,
			token_hash,
			issued_at,
			last_used_at,
			expires_at,
			idle_expires_at,
			revoked_at,
			revoke_reason,
			ip,
			user_agent
		) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?)`,
		params.ID,
		params.UserID,
		params.TokenHash,
		params.IssuedAt.Unix(),
		params.LastUsedAt.Unix(),
		params.ExpiresAt.Unix(),
		params.IdleExpiresAt.Unix(),
		params.IP,
		params.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("users: failed to create refresh session: %w", err)
	}

	return nil
}

func (s *Store) GetRefreshSessionByID(ctx context.Context, id string) (RefreshSession, error) {
	var session RefreshSession
	var revokedAt sql.NullInt64
	var revokeReason sql.NullString
	var ip sql.NullString
	var userAgent sql.NullString

	err := s.db.QueryRowContext(
		ctx,
		`SELECT
			id,
			user_id,
			token_hash,
			issued_at,
			last_used_at,
			expires_at,
			idle_expires_at,
			revoked_at,
			revoke_reason,
			ip,
			user_agent
		FROM auth_refresh_sessions
		WHERE id = ?`,
		id,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		(*unixTimeScanner)(&session.IssuedAt),
		(*unixTimeScanner)(&session.LastUsedAt),
		(*unixTimeScanner)(&session.ExpiresAt),
		(*unixTimeScanner)(&session.IdleExpiresAt),
		&revokedAt,
		&revokeReason,
		&ip,
		&userAgent,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return RefreshSession{}, ErrRefreshSessionNotFound
	}
	if err != nil {
		return RefreshSession{}, fmt.Errorf("users: failed to get refresh session: %w", err)
	}

	session.RevokedAt = nullableUnixTimePointer(revokedAt)
	session.RevokeReason = nullableStringPointer(revokeReason)
	session.IP = nullableStringPointer(ip)
	session.UserAgent = nullableStringPointer(userAgent)
	return session, nil
}

func (s *Store) RotateRefreshSession(ctx context.Context, params RotateRefreshSessionParams) error {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE auth_refresh_sessions
		SET token_hash = ?, last_used_at = ?, idle_expires_at = ?
		WHERE id = ? AND revoked_at IS NULL`,
		params.TokenHash,
		params.LastUsedAt.Unix(),
		params.IdleExpiresAt.Unix(),
		params.ID,
	)
	if err != nil {
		return fmt.Errorf("users: failed to rotate refresh session: %w", err)
	}

	if err := ensureRefreshSessionAffected(result); err != nil {
		return err
	}

	return nil
}

func (s *Store) RevokeRefreshSession(ctx context.Context, id string, reason string, revokedAt time.Time) error {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE auth_refresh_sessions
		SET revoked_at = COALESCE(revoked_at, ?),
			revoke_reason = COALESCE(revoke_reason, ?)
		WHERE id = ?`,
		revokedAt.Unix(),
		reason,
		id,
	)
	if err != nil {
		return fmt.Errorf("users: failed to revoke refresh session: %w", err)
	}

	if err := ensureRefreshSessionAffected(result); err != nil {
		return err
	}

	return nil
}

func (s *Store) RevokeRefreshSessionsByUser(ctx context.Context, userID int64, reason string, revokedAt time.Time) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE auth_refresh_sessions
		SET revoked_at = COALESCE(revoked_at, ?),
			revoke_reason = COALESCE(revoke_reason, ?)
		WHERE user_id = ? AND revoked_at IS NULL`,
		revokedAt.Unix(),
		reason,
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf("users: failed to revoke refresh sessions by user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("users: failed to determine revoked refresh session count: %w", err)
	}

	return rowsAffected, nil
}

func (s *Store) InsertAuthLog(ctx context.Context, params AuthLogParams) error {
	createdAt := params.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO auth_login_logs (
			user_id,
			session_id,
			identifier,
			ip,
			user_agent,
			event_type,
			success,
			failure_reason,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		params.UserID,
		params.SessionID,
		params.Identifier,
		params.IP,
		params.UserAgent,
		params.EventType,
		boolToSQLiteInteger(params.Success),
		params.FailureReason,
		createdAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("users: failed to insert auth log: %w", err)
	}

	return nil
}

func (s *Store) ListAuthLogs(ctx context.Context) ([]AuthLogRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, user_id, session_id, identifier, ip, user_agent, event_type, success, failure_reason, created_at
		FROM auth_login_logs
		ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("users: failed to list auth logs: %w", err)
	}
	defer rows.Close()

	logs := make([]AuthLogRecord, 0)
	for rows.Next() {
		var record AuthLogRecord
		var userID sql.NullInt64
		var sessionID sql.NullString
		var identifier sql.NullString
		var ip sql.NullString
		var userAgent sql.NullString
		var success int64
		var failureReason sql.NullString
		if err := rows.Scan(
			&record.ID,
			&userID,
			&sessionID,
			&identifier,
			&ip,
			&userAgent,
			&record.EventType,
			&success,
			&failureReason,
			(*unixTimeScanner)(&record.CreatedAt),
		); err != nil {
			return nil, fmt.Errorf("users: failed to scan auth log: %w", err)
		}

		record.UserID = nullableInt64Pointer(userID)
		record.SessionID = nullableStringPointer(sessionID)
		record.Identifier = nullableStringPointer(identifier)
		record.IP = nullableStringPointer(ip)
		record.UserAgent = nullableStringPointer(userAgent)
		record.Success = success == 1
		record.FailureReason = nullableStringPointer(failureReason)
		logs = append(logs, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("users: failed to iterate auth logs: %w", err)
	}

	return logs, nil
}

func ensureRefreshSessionAffected(result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("users: failed to determine refresh session rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrRefreshSessionNotFound
	}
	return nil
}

func nullableInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}

	copied := value.Int64
	return &copied
}

type unixTimeScanner time.Time

func (s *unixTimeScanner) Scan(src any) error {
	switch value := src.(type) {
	case int64:
		*s = unixTimeScanner(time.Unix(value, 0).UTC())
		return nil
	case int:
		*s = unixTimeScanner(time.Unix(int64(value), 0).UTC())
		return nil
	case []byte:
		var unixSeconds int64
		if _, err := fmt.Sscan(string(value), &unixSeconds); err != nil {
			return fmt.Errorf("users: invalid unix time %q: %w", string(value), err)
		}
		*s = unixTimeScanner(time.Unix(unixSeconds, 0).UTC())
		return nil
	default:
		return fmt.Errorf("users: unsupported unix time scan type %T", src)
	}
}
