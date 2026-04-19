package systemsettings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const singletonID = 1

type AuthMode string

const (
	AuthModeSingleUser AuthMode = "single_user"
	AuthModeMultiUser  AuthMode = "multi_user"
)

type RegistrationMode string

const (
	RegistrationModeDisabled RegistrationMode = "disabled"
	RegistrationModePublic   RegistrationMode = "public"
)

var (
	ErrInvalidAuthMode         = errors.New("systemsettings: invalid auth mode")
	ErrInvalidRegistrationMode = errors.New("systemsettings: invalid registration mode")
)

type Settings struct {
	AuthMode                          AuthMode         `json:"authMode"`
	RegistrationMode                  RegistrationMode `json:"registrationMode"`
	PasswordLoginEnabled              bool             `json:"passwordLoginEnabled"`
	AdminUserCreateEnabled            bool             `json:"adminUserCreateEnabled"`
	SelfServiceAccountDeletionEnabled bool             `json:"selfServiceAccountDeletionEnabled"`
	UpdatedAt                         time.Time        `json:"updatedAt"`
}

type UpdateInput struct {
	AuthMode                          *AuthMode         `json:"authMode"`
	RegistrationMode                  *RegistrationMode `json:"registrationMode"`
	AdminUserCreateEnabled            *bool             `json:"adminUserCreateEnabled"`
	SelfServiceAccountDeletionEnabled *bool             `json:"selfServiceAccountDeletionEnabled"`
}

type PublicAuthConfig struct {
	AuthMode             AuthMode         `json:"authMode"`
	RegistrationMode     RegistrationMode `json:"registrationMode"`
	PasswordLoginEnabled bool             `json:"passwordLoginEnabled"`
	RegistrationEnabled  bool             `json:"registrationEnabled"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func NewStore(db *sql.DB) *Service {
	return NewService(db)
}

func (s *Service) Get(ctx context.Context) (Settings, error) {
	if s == nil || s.db == nil {
		return Settings{}, errors.New("systemsettings: nil service")
	}

	var settings Settings
	var passwordLoginEnabled int64
	var adminUserCreateEnabled int64
	var selfServiceAccountDeletionEnabled int64
	var updatedAt int64

	err := s.db.QueryRowContext(
		ctx,
		`SELECT
			auth_mode,
			registration_mode,
			password_login_enabled,
			admin_user_create_enabled,
			self_service_account_deletion_enabled,
			updated_at
		FROM system_settings
		WHERE id = ?`,
		singletonID,
	).Scan(
		&settings.AuthMode,
		&settings.RegistrationMode,
		&passwordLoginEnabled,
		&adminUserCreateEnabled,
		&selfServiceAccountDeletionEnabled,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Settings{}, fmt.Errorf("systemsettings: settings row not found: %w", err)
	}
	if err != nil {
		return Settings{}, fmt.Errorf("systemsettings: get settings: %w", err)
	}

	if !settings.AuthMode.Valid() {
		return Settings{}, ErrInvalidAuthMode
	}
	if !settings.RegistrationMode.Valid() {
		return Settings{}, ErrInvalidRegistrationMode
	}

	settings.PasswordLoginEnabled = passwordLoginEnabled == 1
	settings.AdminUserCreateEnabled = adminUserCreateEnabled == 1
	settings.SelfServiceAccountDeletionEnabled = selfServiceAccountDeletionEnabled == 1
	settings.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return settings, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (Settings, error) {
	current, err := s.Get(ctx)
	if err != nil {
		return Settings{}, err
	}

	next := current
	if input.AuthMode != nil {
		next.AuthMode = *input.AuthMode
	}
	if input.RegistrationMode != nil {
		next.RegistrationMode = *input.RegistrationMode
	}
	if input.AdminUserCreateEnabled != nil {
		next.AdminUserCreateEnabled = *input.AdminUserCreateEnabled
	}
	if input.SelfServiceAccountDeletionEnabled != nil {
		next.SelfServiceAccountDeletionEnabled = *input.SelfServiceAccountDeletionEnabled
	}

	if !next.AuthMode.Valid() {
		return Settings{}, ErrInvalidAuthMode
	}
	if !next.RegistrationMode.Valid() {
		return Settings{}, ErrInvalidRegistrationMode
	}

	if next.AuthMode == AuthModeSingleUser {
		next.RegistrationMode = RegistrationModeDisabled
	}

	now := time.Now().UTC().Unix()
	_, err = s.db.ExecContext(
		ctx,
		`UPDATE system_settings
		SET auth_mode = ?,
			registration_mode = ?,
			password_login_enabled = 1,
			admin_user_create_enabled = ?,
			self_service_account_deletion_enabled = ?,
			updated_at = ?
		WHERE id = ?`,
		next.AuthMode,
		next.RegistrationMode,
		boolToSQLiteInteger(next.AdminUserCreateEnabled),
		boolToSQLiteInteger(next.SelfServiceAccountDeletionEnabled),
		now,
		singletonID,
	)
	if err != nil {
		return Settings{}, fmt.Errorf("systemsettings: update settings: %w", err)
	}

	return s.Get(ctx)
}

func (s *Service) PublicConfig(ctx context.Context) (PublicAuthConfig, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return PublicAuthConfig{}, err
	}

	return PublicAuthConfig{
		AuthMode:             settings.AuthMode,
		RegistrationMode:     settings.RegistrationMode,
		PasswordLoginEnabled: settings.PasswordLoginEnabled,
		RegistrationEnabled:  settings.CanPublicRegister(),
	}, nil
}

func (s Settings) CanPublicRegister() bool {
	return s.AuthMode == AuthModeMultiUser && s.RegistrationMode == RegistrationModePublic && s.PasswordLoginEnabled
}

func (s Settings) AllowAdminUserCreate() bool {
	return s.AuthMode == AuthModeMultiUser && s.AdminUserCreateEnabled
}

func (m AuthMode) Valid() bool {
	switch m {
	case AuthModeSingleUser, AuthModeMultiUser:
		return true
	default:
		return false
	}
}

func (m RegistrationMode) Valid() bool {
	switch m {
	case RegistrationModeDisabled, RegistrationModePublic:
		return true
	default:
		return false
	}
}

func boolToSQLiteInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}
