package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const userSelectColumns = `
id,
username,
email,
password_hash,
avatar_path,
role,
bootstrap_password_active,
auth_version,
password_changed_at,
is_banned,
banned_at,
created_at,
updated_at`

var ErrUserNotFound = errors.New("users: user not found")
var ErrUsernameTaken = errors.New("users: username already exists")
var ErrEmailTaken = errors.New("users: email already exists")

const (
	UserStatusActive = "active"
	UserStatusBanned = "banned"
)

type ListParams struct {
	Query    string
	Role     *int
	Status   string
	Sort     string
	Page     int
	PageSize int
}

type ListResult struct {
	Items    []User
	Page     int
	PageSize int
	Total    int
}

type AdminCreateParams struct {
	Username     string
	Email        *string
	PasswordHash string
	Role         int
}

type AdminUpdateParams struct {
	Username string
	Email    *string
	Role     int
}

type Store struct {
	db *sql.DB
}

type CreateParams struct {
	Username                string
	Email                   *string
	PasswordHash            string
	AvatarPath              *string
	Role                    int
	BootstrapPasswordActive bool
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

func (s *Store) Create(ctx context.Context, params CreateParams) (User, error) {
	if s == nil || s.db == nil {
		return User{}, errors.New("users: nil store")
	}

	username := normalizeUsername(params.Username)
	email := normalizeEmailPointer(params.Email)
	now := time.Now().Unix()

	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (
			username,
			email,
			password_hash,
			avatar_path,
			role,
			bootstrap_password_active,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		username,
		email,
		params.PasswordHash,
		params.AvatarPath,
		params.Role,
		boolToSQLiteInteger(params.BootstrapPasswordActive),
		now,
		now,
	)
	if err != nil {
		return User{}, mapWriteError(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("users: failed to read inserted user id: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *Store) AdminCreate(ctx context.Context, params AdminCreateParams) (User, error) {
	return s.Create(ctx, CreateParams{
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: params.PasswordHash,
		Role:         params.Role,
	})
}

func (s *Store) CountSuperAdmins(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = ?`, RoleSuperAdmin).Scan(&count); err != nil {
		return 0, fmt.Errorf("users: failed to count super admins: %w", err)
	}
	return count, nil
}

func (s *Store) GetFirstSuperAdmin(ctx context.Context) (User, error) {
	return s.queryOne(ctx, `SELECT `+userSelectColumns+` FROM users WHERE role = ? ORDER BY id LIMIT 1`, RoleSuperAdmin)
}

func (s *Store) FindByIdentifier(ctx context.Context, identifier string) (User, error) {
	trimmedIdentifier := strings.TrimSpace(identifier)
	if trimmedIdentifier == "" {
		return User{}, ErrUserNotFound
	}

	normalizedEmail := strings.ToLower(trimmedIdentifier)

	return s.queryOne(
		ctx,
		`SELECT `+userSelectColumns+`
		FROM users
		WHERE username = ? COLLATE NOCASE OR email = ? COLLATE NOCASE
		ORDER BY id
		LIMIT 1`,
		trimmedIdentifier,
		normalizedEmail,
	)
}

func (s *Store) GetByID(ctx context.Context, id int64) (User, error) {
	return s.queryOne(ctx, `SELECT `+userSelectColumns+` FROM users WHERE id = ?`, id)
}

func (s *Store) GetAuthVersion(ctx context.Context, id int64) (int64, error) {
	var authVersion int64
	err := s.db.QueryRowContext(ctx, `SELECT auth_version FROM users WHERE id = ?`, id).Scan(&authVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUserNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("users: failed to load auth version: %w", err)
	}

	return authVersion, nil
}

func (s *Store) UpdateProfile(ctx context.Context, id int64, username string, email *string) (User, error) {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE users SET username = ?, email = ?, updated_at = ? WHERE id = ?`,
		normalizeUsername(username),
		normalizeEmailPointer(email),
		time.Now().Unix(),
		id,
	)
	if err != nil {
		return User{}, mapWriteError(err)
	}
	if err := ensureRowsAffected(result); err != nil {
		return User{}, err
	}

	return s.GetByID(ctx, id)
}

func (s *Store) UpdateAvatar(ctx context.Context, id int64, avatarPath *string) (User, error) {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE users SET avatar_path = ?, updated_at = ? WHERE id = ?`,
		avatarPath,
		time.Now().Unix(),
		id,
	)
	if err != nil {
		return User{}, fmt.Errorf("users: failed to update avatar: %w", err)
	}
	if err := ensureRowsAffected(result); err != nil {
		return User{}, err
	}

	return s.GetByID(ctx, id)
}

func (s *Store) UpdatePassword(ctx context.Context, id int64, passwordHash string, bootstrapPasswordActive bool) (User, error) {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE users
		SET password_hash = ?,
			bootstrap_password_active = ?,
			auth_version = auth_version + 1,
			password_changed_at = ?,
			updated_at = ?
		WHERE id = ?`,
		passwordHash,
		boolToSQLiteInteger(bootstrapPasswordActive),
		now.Unix(),
		now.Unix(),
		id,
	)
	if err != nil {
		return User{}, fmt.Errorf("users: failed to update password: %w", err)
	}
	if err := ensureRowsAffected(result); err != nil {
		return User{}, err
	}

	return s.GetByID(ctx, id)
}

func (s *Store) AdminUpdate(ctx context.Context, id int64, params AdminUpdateParams) (User, error) {
	existingUser, err := s.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	authVersionBump := 0
	if existingUser.Role != params.Role {
		authVersionBump = 1
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE users
		SET username = ?,
			email = ?,
			role = ?,
			auth_version = auth_version + ?,
			updated_at = ?
		WHERE id = ?`,
		normalizeUsername(params.Username),
		normalizeEmailPointer(params.Email),
		params.Role,
		authVersionBump,
		time.Now().Unix(),
		id,
	)
	if err != nil {
		return User{}, mapWriteError(err)
	}
	if err := ensureRowsAffected(result); err != nil {
		return User{}, err
	}

	return s.GetByID(ctx, id)
}

func (s *Store) Ban(ctx context.Context, id int64) (User, error) {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE users
		SET is_banned = 1,
			banned_at = COALESCE(banned_at, ?),
			auth_version = auth_version + CASE WHEN is_banned = 0 THEN 1 ELSE 0 END,
			updated_at = ?
		WHERE id = ?`,
		now.Unix(),
		now.Unix(),
		id,
	)
	if err != nil {
		return User{}, fmt.Errorf("users: failed to ban user: %w", err)
	}
	if err := ensureRowsAffected(result); err != nil {
		return User{}, err
	}

	return s.GetByID(ctx, id)
}

func (s *Store) Unban(ctx context.Context, id int64) (User, error) {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE users
		SET is_banned = 0,
			banned_at = NULL,
			auth_version = auth_version + CASE WHEN is_banned = 1 THEN 1 ELSE 0 END,
			updated_at = ?
		WHERE id = ?`,
		now.Unix(),
		id,
	)
	if err != nil {
		return User{}, fmt.Errorf("users: failed to unban user: %w", err)
	}
	if err := ensureRowsAffected(result); err != nil {
		return User{}, err
	}

	return s.GetByID(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("users: failed to delete user: %w", err)
	}

	return ensureRowsAffected(result)
}

func (s *Store) List(ctx context.Context, params ListParams) (ListResult, error) {
	normalized := normalizeListParams(params)

	whereClause, args := buildUserListWhereClause(normalized)

	var total int
	countQuery := `SELECT COUNT(*) FROM users` + whereClause
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("users: failed to count users: %w", err)
	}

	queryArgs := append(append([]any{}, args...), normalized.PageSize, (normalized.Page-1)*normalized.PageSize)
	query := `SELECT ` + userSelectColumns + `
	FROM users` + whereClause + `
	ORDER BY ` + userListOrderByClause(normalized.Sort) + `
	LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return ListResult{}, fmt.Errorf("users: failed to list users: %w", err)
	}
	defer rows.Close()

	items := make([]User, 0, normalized.PageSize)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return ListResult{}, err
		}

		items = append(items, user)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("users: failed to iterate user list: %w", err)
	}

	return ListResult{
		Items:    items,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
		Total:    total,
	}, nil
}

func (s *Store) queryOne(ctx context.Context, query string, args ...any) (User, error) {
	user, err := scanUser(s.db.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func ensureRowsAffected(result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("users: failed to determine affected rows: %w", err)
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

	message := err.Error()
	switch {
	case strings.Contains(message, "UNIQUE constraint failed: users.username"):
		return ErrUsernameTaken
	case strings.Contains(message, "UNIQUE constraint failed: users.email"):
		return ErrEmailTaken
	default:
		return fmt.Errorf("users: write failed: %w", err)
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

	return &normalized
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	copied := value.String
	return &copied
}

func nullableUnixTimePointer(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}

	timestamp := time.Unix(value.Int64, 0).UTC()
	return &timestamp
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner rowScanner) (User, error) {
	var user User
	var email sql.NullString
	var avatarPath sql.NullString
	var bootstrapPasswordActive int64
	var passwordChangedAt sql.NullInt64
	var isBanned int64
	var bannedAt sql.NullInt64

	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&email,
		&user.PasswordHash,
		&avatarPath,
		&user.Role,
		&bootstrapPasswordActive,
		&user.AuthVersion,
		&passwordChangedAt,
		&isBanned,
		&bannedAt,
		(*unixTimeScanner)(&user.CreatedAt),
		(*unixTimeScanner)(&user.UpdatedAt),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("users: query failed: %w", err)
	}

	user.Email = nullableStringPointer(email)
	user.AvatarPath = nullableStringPointer(avatarPath)
	user.BootstrapPasswordActive = bootstrapPasswordActive == 1
	user.PasswordChangedAt = nullableUnixTimePointer(passwordChangedAt)
	user.IsBanned = isBanned == 1
	user.BannedAt = nullableUnixTimePointer(bannedAt)

	return user, nil
}

func normalizeListParams(params ListParams) ListParams {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	switch params.Status {
	case UserStatusActive, UserStatusBanned:
	default:
		params.Status = ""
	}

	switch params.Sort {
	case "created_at_asc", "created_at_desc", "username_asc", "username_desc":
	default:
		params.Sort = "created_at_desc"
	}

	params.Query = strings.TrimSpace(params.Query)
	return params
}

func buildUserListWhereClause(params ListParams) (string, []any) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 5)

	if params.Query != "" {
		pattern := "%" + params.Query + "%"
		clauses = append(clauses, "(username LIKE ? COLLATE NOCASE OR email LIKE ? COLLATE NOCASE)")
		args = append(args, pattern, pattern)
	}
	if params.Role != nil {
		clauses = append(clauses, "role = ?")
		args = append(args, *params.Role)
	}
	switch params.Status {
	case UserStatusActive:
		clauses = append(clauses, "is_banned = 0")
	case UserStatusBanned:
		clauses = append(clauses, "is_banned = 1")
	}

	if len(clauses) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}

func userListOrderByClause(sort string) string {
	switch sort {
	case "created_at_asc":
		return "created_at ASC, id ASC"
	case "username_asc":
		return "username ASC, id ASC"
	case "username_desc":
		return "username DESC, id DESC"
	default:
		return "created_at DESC, id DESC"
	}
}

func boolToSQLiteInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}
