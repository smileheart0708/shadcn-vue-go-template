package accountpolicies

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const singletonID = 1

type Policies struct {
	PublicRegistrationEnabled         bool `json:"publicRegistrationEnabled"`
	SelfServiceAccountDeletionEnabled bool `json:"selfServiceAccountDeletionEnabled"`
}

type UpdateInput struct {
	PublicRegistrationEnabled         *bool `json:"publicRegistrationEnabled"`
	SelfServiceAccountDeletionEnabled *bool `json:"selfServiceAccountDeletionEnabled"`
}

type PublicAuthConfig struct {
	RegistrationEnabled bool `json:"registrationEnabled"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Get(ctx context.Context) (Policies, error) {
	if s == nil || s.db == nil {
		return Policies{}, errors.New("accountpolicies: nil service")
	}

	var policies Policies
	var publicRegistrationEnabled int64
	var selfServiceAccountDeletionEnabled int64

	err := s.db.QueryRowContext(
		ctx,
		`SELECT
			public_registration_enabled,
			self_service_account_deletion_enabled
		FROM account_policies
		WHERE id = ?`,
		singletonID,
	).Scan(&publicRegistrationEnabled, &selfServiceAccountDeletionEnabled)
	if errors.Is(err, sql.ErrNoRows) {
		return Policies{}, fmt.Errorf("accountpolicies: policies row not found: %w", err)
	}
	if err != nil {
		return Policies{}, fmt.Errorf("accountpolicies: get policies: %w", err)
	}

	policies.PublicRegistrationEnabled = publicRegistrationEnabled == 1
	policies.SelfServiceAccountDeletionEnabled = selfServiceAccountDeletionEnabled == 1
	return policies, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (Policies, error) {
	current, err := s.Get(ctx)
	if err != nil {
		return Policies{}, err
	}

	next := current
	if input.PublicRegistrationEnabled != nil {
		next.PublicRegistrationEnabled = *input.PublicRegistrationEnabled
	}
	if input.SelfServiceAccountDeletionEnabled != nil {
		next.SelfServiceAccountDeletionEnabled = *input.SelfServiceAccountDeletionEnabled
	}

	_, err = s.db.ExecContext(
		ctx,
		`UPDATE account_policies
		SET public_registration_enabled = ?,
			self_service_account_deletion_enabled = ?
		WHERE id = ?`,
		boolToSQLiteInteger(next.PublicRegistrationEnabled),
		boolToSQLiteInteger(next.SelfServiceAccountDeletionEnabled),
		singletonID,
	)
	if err != nil {
		return Policies{}, fmt.Errorf("accountpolicies: update policies: %w", err)
	}

	return s.Get(ctx)
}

func (s *Service) PublicAuthConfig(ctx context.Context) (PublicAuthConfig, error) {
	policies, err := s.Get(ctx)
	if err != nil {
		return PublicAuthConfig{}, err
	}

	return PublicAuthConfig{
		RegistrationEnabled: policies.PublicRegistrationEnabled,
	}, nil
}

func boolToSQLiteInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}
