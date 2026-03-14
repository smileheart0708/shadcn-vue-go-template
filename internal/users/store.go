package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrUserNotFound = errors.New("users: user not found")
var ErrUsernameTaken = errors.New("users: username already exists")
var ErrEmailTaken = errors.New("users: email already exists")

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

func (s *Store) CountSuperAdmins(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = ?`, RoleSuperAdmin).Scan(&count); err != nil {
		return 0, fmt.Errorf("users: failed to count super admins: %w", err)
	}
	return count, nil
}

func (s *Store) GetFirstSuperAdmin(ctx context.Context) (User, error) {
	return s.queryOne(ctx, `SELECT id, username, email, password_hash, avatar_path, role, bootstrap_password_active FROM users WHERE role = ? ORDER BY id LIMIT 1`, RoleSuperAdmin)
}

func (s *Store) FindByIdentifier(ctx context.Context, identifier string) (User, error) {
	trimmedIdentifier := strings.TrimSpace(identifier)
	if trimmedIdentifier == "" {
		return User{}, ErrUserNotFound
	}

	normalizedEmail := strings.ToLower(trimmedIdentifier)

	return s.queryOne(
		ctx,
		`SELECT id, username, email, password_hash, avatar_path, role, bootstrap_password_active
		FROM users
		WHERE username = ? COLLATE NOCASE OR email = ? COLLATE NOCASE
		ORDER BY id
		LIMIT 1`,
		trimmedIdentifier,
		normalizedEmail,
	)
}

func (s *Store) GetByID(ctx context.Context, id int64) (User, error) {
	return s.queryOne(ctx, `SELECT id, username, email, password_hash, avatar_path, role, bootstrap_password_active FROM users WHERE id = ?`, id)
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
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE users SET password_hash = ?, bootstrap_password_active = ?, updated_at = ? WHERE id = ?`,
		passwordHash,
		boolToSQLiteInteger(bootstrapPasswordActive),
		time.Now().Unix(),
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

func (s *Store) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("users: failed to delete user: %w", err)
	}

	return ensureRowsAffected(result)
}

func (s *Store) queryOne(ctx context.Context, query string, args ...any) (User, error) {
	var user User
	var email sql.NullString
	var avatarPath sql.NullString
	var bootstrapPasswordActive int64

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.Username,
		&email,
		&user.PasswordHash,
		&avatarPath,
		&user.Role,
		&bootstrapPasswordActive,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("users: query failed: %w", err)
	}

	user.Email = nullableStringPointer(email)
	user.AvatarPath = nullableStringPointer(avatarPath)
	user.BootstrapPasswordActive = bootstrapPasswordActive == 1

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

func boolToSQLiteInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}
