package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"main/internal/database"
	"main/internal/users"
)

func TestRefreshSessionDetectsStaleCachedTokenHashDuringRotation(t *testing.T) {
	t.Parallel()

	container, err := database.Open(context.Background(), database.Options{
		Path: filepath.Join(t.TempDir(), "app.db"),
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	})

	store := users.NewStore(container.DB())
	user, err := store.Create(context.Background(), users.CreateParams{
		Username:     "member",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleUser,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	service := NewService(Options{
		Issuer:             "test-suite",
		Secret:             []byte("test-secret"),
		TTL:                time.Hour,
		RefreshIdleTTL:     7 * 24 * time.Hour,
		RefreshAbsoluteTTL: 30 * 24 * time.Hour,
	}, store)

	sessionID, refreshToken, err := service.CreateRefreshSession(context.Background(), user, "", "")
	if err != nil {
		t.Fatalf("failed to create refresh session: %v", err)
	}

	state, err := service.LoadRefreshSessionState(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("failed to prime refresh session cache: %v", err)
	}

	nextRefreshToken := composeRefreshToken(sessionID, "other-secret")
	if err := store.RotateRefreshSession(context.Background(), users.RotateRefreshSessionParams{
		ID:                sessionID,
		PreviousTokenHash: state.TokenHash,
		TokenHash:         hashRefreshToken(nextRefreshToken),
		LastUsedAt:        time.Now().UTC(),
		IdleExpiresAt:     time.Now().UTC().Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("failed to simulate competing refresh rotation: %v", err)
	}

	_, attempt, err := service.RefreshSession(context.Background(), refreshToken)
	if !errors.Is(err, ErrTokenReuseDetected) {
		t.Fatalf("expected ErrTokenReuseDetected, got %v", err)
	}
	if attempt.FailureReason != "token_reuse_detected" {
		t.Fatalf("expected token_reuse_detected failure reason, got %q", attempt.FailureReason)
	}

	session, err := store.GetRefreshSessionByID(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("failed to load revoked refresh session: %v", err)
	}
	if session.RevokedAt == nil {
		t.Fatal("expected stale-token reuse to revoke the refresh session")
	}
	if session.RevokeReason == nil || *session.RevokeReason != "token_reuse_detected" {
		t.Fatalf("expected revoke reason token_reuse_detected, got %+v", session.RevokeReason)
	}
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()

	passwordHash, err := users.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return passwordHash
}
