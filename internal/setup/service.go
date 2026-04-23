package setup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"main/internal/audit"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/database"
	"main/internal/identity"
)

const (
	StatePending   = "pending"
	StateCompleted = "completed"
)

var ErrAlreadyCompleted = errors.New("setup: install already completed")

type State struct {
	SetupState       string     `json:"setupState"`
	SetupCompleted   bool       `json:"setupCompleted"`
	OwnerUserID      *int64     `json:"ownerUserId"`
	SetupCompletedAt *time.Time `json:"setupCompletedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type CompleteSetupInput struct {
	Username string
	Password string
}

type Service struct {
	db         *sql.DB
	identities *identity.Service
}

func NewService(db *sql.DB, identities *identity.Service) *Service {
	return &Service{db: db, identities: identities}
}

func (s *Service) GetState(ctx context.Context) (State, error) {
	if s == nil || s.db == nil {
		return State{}, errors.New("setup: nil service")
	}

	var state State
	var ownerUserID sql.NullInt64
	var setupCompletedAt sql.NullInt64
	var createdAt int64
	var updatedAt int64

	err := s.db.QueryRowContext(
		ctx,
		`SELECT setup_state, owner_user_id, setup_completed_at, created_at, updated_at
		FROM install_state
		WHERE id = 1`,
	).Scan(&state.SetupState, &ownerUserID, &setupCompletedAt, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return State{}, fmt.Errorf("setup: install state row missing: %w", err)
	}
	if err != nil {
		return State{}, fmt.Errorf("setup: get install state: %w", err)
	}

	state.SetupCompleted = state.SetupState == StateCompleted
	if ownerUserID.Valid {
		state.OwnerUserID = &ownerUserID.Int64
	}
	if setupCompletedAt.Valid {
		next := time.Unix(setupCompletedAt.Int64, 0).UTC()
		state.SetupCompletedAt = &next
	}
	state.CreatedAt = time.Unix(createdAt, 0).UTC()
	state.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return state, nil
}

func (s *Service) Complete(ctx context.Context, input CompleteSetupInput, requestAudit identity.ActionAuditContext) (identity.UserWithRoles, error) {
	if s == nil || s.db == nil || s.identities == nil {
		return identity.UserWithRoles{}, errors.New("setup: nil service")
	}

	username := strings.TrimSpace(input.Username)
	if username == "" {
		return identity.UserWithRoles{}, fmt.Errorf("setup: username is required")
	}
	password := strings.TrimSpace(input.Password)
	if len(password) < auth.MinPasswordLength {
		return identity.UserWithRoles{}, auth.ErrPasswordTooShort
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return identity.UserWithRoles{}, fmt.Errorf("setup: hash owner password: %w", err)
	}

	var ownerUserID int64
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var setupState string
		if err := tx.QueryRowContext(ctx, `SELECT setup_state FROM install_state WHERE id = 1`).Scan(&setupState); err != nil {
			return fmt.Errorf("setup: load install state: %w", err)
		}
		if setupState == StateCompleted {
			return ErrAlreadyCompleted
		}

		now := time.Now().UTC()
		result, err := tx.ExecContext(
			ctx,
			`INSERT INTO users (
				username,
				email,
				avatar_path,
				status,
				role,
				security_version,
				disabled_at,
				created_at,
				updated_at
			) VALUES (?, ?, NULL, ?, ?, 1, NULL, ?, ?)`,
			username,
			nil,
			identity.StatusActive,
			authorization.RoleOwner,
			now.Unix(),
			now.Unix(),
		)
		if err != nil {
			return mapIdentityWriteError(err)
		}

		ownerUserID, err = result.LastInsertId()
		if err != nil {
			return fmt.Errorf("setup: read owner user id: %w", err)
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
			ownerUserID,
			passwordHash,
			now.Unix(),
			now.Unix(),
			now.Unix(),
		); err != nil {
			return fmt.Errorf("setup: insert owner credential: %w", err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`UPDATE install_state
			SET setup_state = ?,
				owner_user_id = ?,
				setup_completed_at = ?,
				updated_at = ?
			WHERE id = 1`,
			StateCompleted,
			ownerUserID,
			now.Unix(),
			now.Unix(),
		); err != nil {
			return fmt.Errorf("setup: update install state: %w", err)
		}
		return audit.NewService(tx).Log(ctx, audit.Entry{
			ActorUserID:   new(ownerUserID),
			SubjectUserID: new(ownerUserID),
			AuthSessionID: requestAudit.AuthSessionID,
			EventType:     audit.EventSetupCompleted,
			Outcome:       audit.OutcomeSuccess,
			IP:            requestAudit.IP,
			UserAgent:     requestAudit.UserAgent,
			OccurredAt:    now,
		})
	})
	if err != nil {
		return identity.UserWithRoles{}, err
	}

	return s.identities.GetUserByID(ctx, ownerUserID)
}

func mapIdentityWriteError(err error) error {
	switch {
	case errors.Is(err, identity.ErrUsernameTaken),
		errors.Is(err, identity.ErrOwnerAlreadyExists):
		return err
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed: users.username"):
		return identity.ErrUsernameTaken
	case strings.Contains(message, "unique constraint failed: users.role"):
		return identity.ErrOwnerAlreadyExists
	default:
		return fmt.Errorf("setup: write failed: %w", err)
	}
}
