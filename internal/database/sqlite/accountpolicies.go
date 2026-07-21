package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"main/internal/accountpolicies"
)

const accountPoliciesSingletonID = 1

func (s *Store) GetPolicies(ctx context.Context) (accountpolicies.Policies, error) {
	db, err := s.requireDB()
	if err != nil {
		return accountpolicies.Policies{}, err
	}

	var policies accountpolicies.Policies
	var registrationEnabled int64
	var selfServiceDeletionEnabled int64
	err = db.QueryRowContext(
		ctx,
		`SELECT public_registration_enabled, self_service_account_deletion_enabled
		FROM account_policies
		WHERE id = ?`,
		accountPoliciesSingletonID,
	).Scan(&registrationEnabled, &selfServiceDeletionEnabled)
	if errors.Is(err, sql.ErrNoRows) {
		return accountpolicies.Policies{}, errors.New("accountpolicies: policies row not found")
	}
	if err != nil {
		return accountpolicies.Policies{}, fmt.Errorf("accountpolicies: get policies: %w", err)
	}
	policies.PublicRegistrationEnabled = registrationEnabled == 1
	policies.SelfServiceAccountDeletionEnabled = selfServiceDeletionEnabled == 1
	return policies, nil
}

func (s *Store) UpdatePolicies(ctx context.Context, input accountpolicies.UpdateInput) (accountpolicies.Policies, error) {
	db, err := s.requireDB()
	if err != nil {
		return accountpolicies.Policies{}, err
	}

	registrationEnabled := nullableBoolInteger(input.PublicRegistrationEnabled)
	selfServiceDeletionEnabled := nullableBoolInteger(input.SelfServiceAccountDeletionEnabled)
	result, err := db.ExecContext(
		ctx,
		`UPDATE account_policies
		SET public_registration_enabled = COALESCE(?, public_registration_enabled),
			self_service_account_deletion_enabled = COALESCE(?, self_service_account_deletion_enabled)
		WHERE id = ?`,
		registrationEnabled,
		selfServiceDeletionEnabled,
		accountPoliciesSingletonID,
	)
	if err != nil {
		return accountpolicies.Policies{}, fmt.Errorf("accountpolicies: update policies: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return accountpolicies.Policies{}, fmt.Errorf("accountpolicies: update policies rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return accountpolicies.Policies{}, errors.New("accountpolicies: policies row not found")
	}
	return s.GetPolicies(ctx)
}

func nullableBoolInteger(value *bool) any {
	if value == nil {
		return nil
	}
	if *value {
		return 1
	}
	return 0
}
