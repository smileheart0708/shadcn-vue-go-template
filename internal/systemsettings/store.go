package systemsettings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const singletonID = 1

type RegistrationMode string

const (
	RegistrationModeDisabled RegistrationMode = "disabled"
	RegistrationModePassword RegistrationMode = "password"
)

var ErrInvalidRegistrationMode = errors.New("systemsettings: invalid registration mode")

type Settings struct {
	RegistrationMode RegistrationMode
	UpdatedAt        time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Get(ctx context.Context) (Settings, error) {
	if s == nil || s.db == nil {
		return Settings{}, errors.New("systemsettings: nil store")
	}

	var settings Settings
	var updatedAt int64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT registration_mode, updated_at FROM system_settings WHERE id = ?`,
		singletonID,
	).Scan(&settings.RegistrationMode, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Settings{}, fmt.Errorf("systemsettings: settings row not found: %w", err)
	}
	if err != nil {
		return Settings{}, fmt.Errorf("systemsettings: failed to load settings: %w", err)
	}
	if !settings.RegistrationMode.Valid() {
		return Settings{}, ErrInvalidRegistrationMode
	}

	settings.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return settings, nil
}

func (s *Store) UpdateRegistrationMode(ctx context.Context, mode RegistrationMode) (Settings, error) {
	if s == nil || s.db == nil {
		return Settings{}, errors.New("systemsettings: nil store")
	}
	if !mode.Valid() {
		return Settings{}, ErrInvalidRegistrationMode
	}

	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO system_settings (id, registration_mode, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			registration_mode = excluded.registration_mode,
			updated_at = excluded.updated_at`,
		singletonID,
		mode,
		now,
	)
	if err != nil {
		return Settings{}, fmt.Errorf("systemsettings: failed to update registration mode: %w", err)
	}

	return s.Get(ctx)
}

func (m RegistrationMode) Valid() bool {
	switch m {
	case RegistrationModeDisabled, RegistrationModePassword:
		return true
	default:
		return false
	}
}
