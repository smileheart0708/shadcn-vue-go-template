package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"main/internal/audit"
	"main/internal/authorization"
	"main/internal/database"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

var (
	ErrUserNotFound       = errors.New("identity: user not found")
	ErrUsernameTaken      = errors.New("identity: username already exists")
	ErrEmailTaken         = errors.New("identity: email already exists")
	ErrOwnerAlreadyExists = errors.New("identity: owner already exists")
)

type User struct {
	ID              int64
	Username        string
	Email           *string
	AvatarPath      *string
	Status          string
	SecurityVersion int64
	DisabledAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type UserWithRoles struct {
	User
	RoleKeys []string
}

type AuthRecord struct {
	UserWithRoles
	PasswordHash      string
	PasswordChangedAt *time.Time
}

type CreateUserParams struct {
	Username         string
	Email            *string
	PasswordHash     string
	RoleKey          string
	AssignedByUserID *int64
}

type UpdateManagedUserParams struct {
	Username string
	Email    *string
	RoleKey  string
}

type ListUsersParams struct {
	Query    string
	RoleKey  string
	Status   string
	Page     int
	PageSize int
}

type ListUsersResult struct {
	Items    []UserWithRoles
	Page     int
	PageSize int
	Total    int
}

type ActionAuditContext struct {
	ActorUserID   *int64
	AuthSessionID *string
	IP            *string
	UserAgent     *string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

func (s *Service) CreateUser(ctx context.Context, params CreateUserParams, action ActionAuditContext) (UserWithRoles, error) {
	if s == nil || s.db == nil {
		return UserWithRoles{}, errors.New("identity: nil service")
	}

	username := normalizeUsername(params.Username)
	if username == "" {
		return UserWithRoles{}, fmt.Errorf("identity: username is required")
	}
	roleKey := strings.TrimSpace(params.RoleKey)
	if roleKey == "" {
		return UserWithRoles{}, fmt.Errorf("identity: role key is required")
	}
	passwordHash := strings.TrimSpace(params.PasswordHash)
	if passwordHash == "" {
		return UserWithRoles{}, fmt.Errorf("identity: password hash is required")
	}

	var created UserWithRoles
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		result, err := tx.ExecContext(
			ctx,
			`INSERT INTO users (
				username,
				email,
				avatar_path,
				status,
				security_version,
				disabled_at,
				created_at,
				updated_at
			) VALUES (?, ?, NULL, ?, 1, NULL, ?, ?)`,
			username,
			normalizeEmailPointer(params.Email),
			StatusActive,
			now.Unix(),
			now.Unix(),
		)
		if err != nil {
			return mapWriteError(err)
		}

		userID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("identity: read inserted user id: %w", err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO credentials (
				user_id,
				password_hash,
				password_changed_at,
				created_at,
				updated_at
			) VALUES (?, ?, ?, ?, ?)`,
			userID,
			passwordHash,
			now.Unix(),
			now.Unix(),
			now.Unix(),
		); err != nil {
			return fmt.Errorf("identity: insert credential: %w", err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO user_roles (user_id, role_key, assigned_at, assigned_by_user_id)
			VALUES (?, ?, ?, ?)`,
			userID,
			roleKey,
			now.Unix(),
			params.AssignedByUserID,
		); err != nil {
			return mapWriteError(err)
		}

		if action.ActorUserID != nil {
			if err := audit.NewService(tx).Log(ctx, audit.Entry{
				ActorUserID:   action.ActorUserID,
				SubjectUserID: new(userID),
				AuthSessionID: action.AuthSessionID,
				EventType:     audit.EventUserCreated,
				Outcome:       audit.OutcomeSuccess,
				IP:            action.IP,
				UserAgent:     action.UserAgent,
				Metadata: map[string]any{
					"roleKeys": []string{roleKey},
				},
				OccurredAt: now,
			}); err != nil {
				return err
			}
		}

		created, err = s.getUserWithRolesQuerier(ctx, tx, userID)
		return err
	})
	if err != nil {
		return UserWithRoles{}, err
	}

	return created, nil
}

func (s *Service) GetUserByID(ctx context.Context, userID int64) (UserWithRoles, error) {
	if s == nil || s.db == nil {
		return UserWithRoles{}, errors.New("identity: nil service")
	}
	return s.getUserWithRolesQuerier(ctx, s.db, userID)
}

func (s *Service) GetAuthRecordByID(ctx context.Context, userID int64) (AuthRecord, error) {
	if s == nil || s.db == nil {
		return AuthRecord{}, errors.New("identity: nil service")
	}
	return getAuthRecord(ctx, s.db, userIDSelectQuery(), userID)
}

func (s *Service) GetAuthRecordByIdentifier(ctx context.Context, identifier string) (AuthRecord, error) {
	if s == nil || s.db == nil {
		return AuthRecord{}, errors.New("identity: nil service")
	}

	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return AuthRecord{}, ErrUserNotFound
	}

	return getAuthRecord(
		ctx,
		s.db,
		`SELECT
			u.id,
			u.username,
			u.email,
			u.avatar_path,
			u.status,
			u.security_version,
			u.disabled_at,
			u.created_at,
			u.updated_at,
			c.password_hash,
			c.password_changed_at,
			ur.role_key
		FROM users u
		INNER JOIN credentials c ON c.user_id = u.id
		INNER JOIN user_roles ur ON ur.user_id = u.id
		WHERE u.username = ? COLLATE NOCASE OR u.email = ? COLLATE NOCASE
		ORDER BY u.id
		LIMIT 1`,
		trimmed,
		strings.ToLower(trimmed),
	)
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, username string, email *string) (UserWithRoles, error) {
	if s == nil || s.db == nil {
		return UserWithRoles{}, errors.New("identity: nil service")
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE users
		SET username = ?, email = ?, updated_at = ?
		WHERE id = ?`,
		normalizeUsername(username),
		normalizeEmailPointer(email),
		time.Now().UTC().Unix(),
		userID,
	)
	if err != nil {
		return UserWithRoles{}, mapWriteError(err)
	}
	if err := ensureRowsAffected(result); err != nil {
		return UserWithRoles{}, err
	}

	return s.GetUserByID(ctx, userID)
}

func (s *Service) UpdateAvatar(ctx context.Context, userID int64, avatarPath *string) (UserWithRoles, error) {
	if s == nil || s.db == nil {
		return UserWithRoles{}, errors.New("identity: nil service")
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE users
		SET avatar_path = ?, updated_at = ?
		WHERE id = ?`,
		avatarPath,
		time.Now().UTC().Unix(),
		userID,
	)
	if err != nil {
		return UserWithRoles{}, fmt.Errorf("identity: update avatar: %w", err)
	}
	if err := ensureRowsAffected(result); err != nil {
		return UserWithRoles{}, err
	}

	return s.GetUserByID(ctx, userID)
}

func (s *Service) UpdateManagedUser(ctx context.Context, userID int64, params UpdateManagedUserParams, action ActionAuditContext) (UserWithRoles, error) {
	if s == nil || s.db == nil {
		return UserWithRoles{}, errors.New("identity: nil service")
	}

	current, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return UserWithRoles{}, err
	}

	nextRoleKey := strings.TrimSpace(params.RoleKey)
	if nextRoleKey == "" {
		nextRoleKey = firstRoleKey(current.RoleKeys)
	}

	var updated UserWithRoles
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		roleChanged := nextRoleKey != firstRoleKey(current.RoleKeys)

		result, err := tx.ExecContext(
			ctx,
			`UPDATE users
			SET username = ?, email = ?, security_version = security_version + ?, updated_at = ?
			WHERE id = ?`,
			normalizeUsername(params.Username),
			normalizeEmailPointer(params.Email),
			boolToInt(roleChanged),
			now.Unix(),
			userID,
		)
		if err != nil {
			return mapWriteError(err)
		}
		if err := ensureRowsAffected(result); err != nil {
			return err
		}

		if roleChanged {
			if _, err := tx.ExecContext(
				ctx,
				`UPDATE user_roles
				SET role_key = ?, assigned_at = ?, assigned_by_user_id = ?
				WHERE user_id = ?`,
				nextRoleKey,
				now.Unix(),
				action.ActorUserID,
				userID,
			); err != nil {
				return mapWriteError(err)
			}

			if err := revokeUserSessionsTx(ctx, tx, userID, "role_changed", now); err != nil {
				return err
			}

			if err := audit.NewService(tx).Log(ctx, audit.Entry{
				ActorUserID:   action.ActorUserID,
				SubjectUserID: new(userID),
				AuthSessionID: action.AuthSessionID,
				EventType:     audit.EventRoleChanged,
				Outcome:       audit.OutcomeSuccess,
				IP:            action.IP,
				UserAgent:     action.UserAgent,
				Metadata: map[string]any{
					"previousRoleKeys": current.RoleKeys,
					"roleKeys":         []string{nextRoleKey},
				},
				OccurredAt: now,
			}); err != nil {
				return err
			}
		}

		updated, err = s.getUserWithRolesQuerier(ctx, tx, userID)
		return err
	})
	if err != nil {
		return UserWithRoles{}, err
	}

	return updated, nil
}

func (s *Service) SetUserStatus(ctx context.Context, userID int64, status string, action ActionAuditContext) (UserWithRoles, error) {
	if s == nil || s.db == nil {
		return UserWithRoles{}, errors.New("identity: nil service")
	}

	current, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return UserWithRoles{}, err
	}

	status = normalizeStatus(status)
	if status == "" {
		return UserWithRoles{}, fmt.Errorf("identity: invalid status")
	}
	if current.Status == status {
		return current, nil
	}

	var updated UserWithRoles
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		var disabledAt any
		if status == StatusDisabled {
			disabledAt = now.Unix()
		}

		result, err := tx.ExecContext(
			ctx,
			`UPDATE users
			SET status = ?,
				disabled_at = ?,
				security_version = security_version + 1,
				updated_at = ?
			WHERE id = ?`,
			status,
			disabledAt,
			now.Unix(),
			userID,
		)
		if err != nil {
			return fmt.Errorf("identity: update status: %w", err)
		}
		if err := ensureRowsAffected(result); err != nil {
			return err
		}

		if err := revokeUserSessionsTx(ctx, tx, userID, "status_changed", now); err != nil {
			return err
		}

		eventType := audit.EventUserDisabled
		if status == StatusActive {
			eventType = audit.EventUserEnabled
		}

		if err := audit.NewService(tx).Log(ctx, audit.Entry{
			ActorUserID:   action.ActorUserID,
			SubjectUserID: new(userID),
			AuthSessionID: action.AuthSessionID,
			EventType:     eventType,
			Outcome:       audit.OutcomeSuccess,
			IP:            action.IP,
			UserAgent:     action.UserAgent,
			OccurredAt:    now,
		}); err != nil {
			return err
		}

		updated, err = s.getUserWithRolesQuerier(ctx, tx, userID)
		return err
	})
	if err != nil {
		return UserWithRoles{}, err
	}

	return updated, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID int64, action ActionAuditContext, reason string) error {
	if s == nil || s.db == nil {
		return errors.New("identity: nil service")
	}

	if _, err := s.GetUserByID(ctx, userID); err != nil {
		return err
	}

	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		result, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID)
		if err != nil {
			return fmt.Errorf("identity: delete user: %w", err)
		}
		if err := ensureRowsAffected(result); err != nil {
			return err
		}

		return audit.NewService(tx).Log(ctx, audit.Entry{
			ActorUserID:   action.ActorUserID,
			SubjectUserID: new(userID),
			AuthSessionID: action.AuthSessionID,
			EventType:     audit.EventAccountDeleted,
			Outcome:       audit.OutcomeSuccess,
			Reason:        stringPointerOrNil(reason),
			IP:            action.IP,
			UserAgent:     action.UserAgent,
			OccurredAt:    now,
		})
	})
}

func (s *Service) ListUsers(ctx context.Context, params ListUsersParams) (ListUsersResult, error) {
	if s == nil || s.db == nil {
		return ListUsersResult{}, errors.New("identity: nil service")
	}

	params = normalizeListParams(params)
	whereClause, args := buildListWhere(params)

	var total int
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		FROM users u
		INNER JOIN user_roles ur ON ur.user_id = u.id`+whereClause,
		args...,
	).Scan(&total); err != nil {
		return ListUsersResult{}, fmt.Errorf("identity: count users: %w", err)
	}

	queryArgs := append(append([]any{}, args...), params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
			u.id,
			u.username,
			u.email,
			u.avatar_path,
			u.status,
			u.security_version,
			u.disabled_at,
			u.created_at,
			u.updated_at,
			ur.role_key
		FROM users u
		INNER JOIN user_roles ur ON ur.user_id = u.id`+whereClause+`
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		return ListUsersResult{}, fmt.Errorf("identity: list users: %w", err)
	}
	defer rows.Close()

	items := make([]UserWithRoles, 0, params.PageSize)
	for rows.Next() {
		item, err := scanUserWithRole(rows)
		if err != nil {
			return ListUsersResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListUsersResult{}, fmt.Errorf("identity: iterate users: %w", err)
	}

	return ListUsersResult{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (s *Service) getUserWithRolesQuerier(ctx context.Context, db database.DBTX, userID int64) (UserWithRoles, error) {
	row := db.QueryRowContext(
		ctx,
		`SELECT
			u.id,
			u.username,
			u.email,
			u.avatar_path,
			u.status,
			u.security_version,
			u.disabled_at,
			u.created_at,
			u.updated_at,
			ur.role_key
		FROM users u
		INNER JOIN user_roles ur ON ur.user_id = u.id
		WHERE u.id = ?`,
		userID,
	)
	return scanUserWithRole(row)
}

func getAuthRecord(ctx context.Context, db database.DBTX, query string, args ...any) (AuthRecord, error) {
	var record AuthRecord
	var email sql.NullString
	var avatarPath sql.NullString
	var disabledAt sql.NullInt64
	var passwordChangedAt sql.NullInt64
	var createdAt int64
	var updatedAt int64
	var roleKey string

	err := db.QueryRowContext(ctx, query, args...).Scan(
		&record.ID,
		&record.Username,
		&email,
		&avatarPath,
		&record.Status,
		&record.SecurityVersion,
		&disabledAt,
		&createdAt,
		&updatedAt,
		&record.PasswordHash,
		&passwordChangedAt,
		&roleKey,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return AuthRecord{}, ErrUserNotFound
	}
	if err != nil {
		return AuthRecord{}, fmt.Errorf("identity: get auth record: %w", err)
	}

	record.Email = nullableStringPointer(email)
	record.AvatarPath = nullableStringPointer(avatarPath)
	record.DisabledAt = nullableUnixTimePointer(disabledAt)
	record.PasswordChangedAt = nullableUnixTimePointer(passwordChangedAt)
	record.CreatedAt = time.Unix(createdAt, 0).UTC()
	record.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	record.RoleKeys = []string{roleKey}
	return record, nil
}

func userIDSelectQuery() string {
	return `SELECT
		u.id,
		u.username,
		u.email,
		u.avatar_path,
		u.status,
		u.security_version,
		u.disabled_at,
		u.created_at,
		u.updated_at,
		c.password_hash,
		c.password_changed_at,
		ur.role_key
	FROM users u
	INNER JOIN credentials c ON c.user_id = u.id
	INNER JOIN user_roles ur ON ur.user_id = u.id
	WHERE u.id = ?`
}

func scanUserWithRole(scanner interface{ Scan(dest ...any) error }) (UserWithRoles, error) {
	var user UserWithRoles
	var email sql.NullString
	var avatarPath sql.NullString
	var disabledAt sql.NullInt64
	var createdAt int64
	var updatedAt int64
	var roleKey string

	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&email,
		&avatarPath,
		&user.Status,
		&user.SecurityVersion,
		&disabledAt,
		&createdAt,
		&updatedAt,
		&roleKey,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return UserWithRoles{}, ErrUserNotFound
	}
	if err != nil {
		return UserWithRoles{}, fmt.Errorf("identity: scan user: %w", err)
	}

	user.Email = nullableStringPointer(email)
	user.AvatarPath = nullableStringPointer(avatarPath)
	user.DisabledAt = nullableUnixTimePointer(disabledAt)
	user.CreatedAt = time.Unix(createdAt, 0).UTC()
	user.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	user.RoleKeys = []string{roleKey}
	return user, nil
}

func revokeUserSessionsTx(ctx context.Context, tx *sql.Tx, userID int64, reason string, now time.Time) error {
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, ?),
			revoke_reason = COALESCE(revoke_reason, ?)
		WHERE user_id = ? AND revoked_at IS NULL`,
		now.Unix(),
		strings.TrimSpace(reason),
		userID,
	); err != nil {
		return fmt.Errorf("identity: revoke sessions: %w", err)
	}
	return nil
}

func buildListWhere(params ListUsersParams) (string, []any) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 4)

	if params.Query != "" {
		pattern := "%" + params.Query + "%"
		clauses = append(clauses, "(u.username LIKE ? COLLATE NOCASE OR u.email LIKE ? COLLATE NOCASE)")
		args = append(args, pattern, pattern)
	}
	if params.RoleKey != "" {
		clauses = append(clauses, "ur.role_key = ?")
		args = append(args, params.RoleKey)
	}
	if params.Status != "" {
		clauses = append(clauses, "u.status = ?")
		args = append(args, params.Status)
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func normalizeListParams(params ListUsersParams) ListUsersParams {
	params.Query = strings.TrimSpace(params.Query)
	params.RoleKey = strings.TrimSpace(params.RoleKey)
	params.Status = normalizeStatus(params.Status)
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	return params
}

func normalizeStatus(status string) string {
	switch strings.TrimSpace(status) {
	case StatusActive:
		return StatusActive
	case StatusDisabled:
		return StatusDisabled
	default:
		return ""
	}
}

func normalizeUsername(value string) string {
	return strings.TrimSpace(value)
}

func normalizeEmailPointer(email *string) *string {
	if email == nil {
		return nil
	}

	normalized := strings.ToLower(strings.TrimSpace(*email))
	if normalized == "" {
		return nil
	}
	return new(normalized)
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return new(value.String)
}

func nullableUnixTimePointer(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	next := time.Unix(value.Int64, 0).UTC()
	return &next
}

func ensureRowsAffected(result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("identity: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func mapWriteError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed: users.username"):
		return ErrUsernameTaken
	case strings.Contains(message, "unique constraint failed: users.email"):
		return ErrEmailTaken
	case strings.Contains(message, "unique constraint failed: user_roles.role_key"):
		return ErrOwnerAlreadyExists
	default:
		return fmt.Errorf("identity: write failed: %w", err)
	}
}

func firstRoleKey(roleKeys []string) string {
	if len(roleKeys) == 0 {
		return ""
	}
	return roleKeys[0]
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func stringPointerOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return new(value)
}

func AvatarDir(dataDir string) string {
	return filepath.Join(dataDir, "avatars")
}

func IsManageableRoleKey(roleKey string) bool {
	switch roleKey {
	case authorization.RoleAdmin, authorization.RoleUser:
		return true
	default:
		return false
	}
}
