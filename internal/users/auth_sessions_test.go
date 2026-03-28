package users

import (
	"context"
	"testing"
	"time"
)

func TestInsertAuthLogDropsMissingSessionReferenceButKeepsUser(t *testing.T) {
	t.Parallel()

	store, _, _ := newBootstrapTestStore(t)
	user, err := store.Create(context.Background(), CreateParams{
		Username:     "member",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         RoleUser,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	sessionID := "missing-session"
	createdAt := time.Unix(1_700_000_000, 0).UTC()
	if err := store.InsertAuthLog(context.Background(), AuthLogParams{
		UserID:        new(user.ID),
		SessionID:     new(sessionID),
		EventType:     "refresh_failed",
		Success:       false,
		FailureReason: new("refresh_session_not_found"),
		CreatedAt:     createdAt,
	}); err != nil {
		t.Fatalf("expected auth log insert to recover from stale session reference: %v", err)
	}

	logs, err := store.ListAuthLogs(context.Background())
	if err != nil {
		t.Fatalf("failed to list auth logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 auth log, got %d", len(logs))
	}
	if logs[0].UserID == nil || *logs[0].UserID != user.ID {
		t.Fatalf("expected auth log to preserve user id %d, got %+v", user.ID, logs[0].UserID)
	}
	if logs[0].SessionID != nil {
		t.Fatalf("expected stale session id to be dropped, got %+v", logs[0].SessionID)
	}
	if logs[0].CreatedAt != createdAt {
		t.Fatalf("expected created_at %v, got %v", createdAt, logs[0].CreatedAt)
	}
}

func TestInsertAuthLogDropsDeletedUserAndSessionReferences(t *testing.T) {
	t.Parallel()

	store, _, _ := newBootstrapTestStore(t)
	user, err := store.Create(context.Background(), CreateParams{
		Username:     "member",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         RoleUser,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	sessionID := "session-1"
	now := time.Now().UTC()
	if err := store.CreateRefreshSession(context.Background(), CreateRefreshSessionParams{
		ID:            sessionID,
		UserID:        user.ID,
		TokenHash:     "hash",
		IssuedAt:      now,
		LastUsedAt:    now,
		ExpiresAt:     now.Add(time.Hour),
		IdleExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("failed to create refresh session: %v", err)
	}
	if err := store.Delete(context.Background(), user.ID); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	if err := store.InsertAuthLog(context.Background(), AuthLogParams{
		UserID:        new(user.ID),
		SessionID:     new(sessionID),
		EventType:     "refresh_failed",
		Success:       false,
		FailureReason: new("user_not_found"),
	}); err != nil {
		t.Fatalf("expected auth log insert to recover from deleted user/session references: %v", err)
	}

	logs, err := store.ListAuthLogs(context.Background())
	if err != nil {
		t.Fatalf("failed to list auth logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 auth log, got %d", len(logs))
	}
	if logs[0].UserID != nil || logs[0].SessionID != nil {
		t.Fatalf("expected deleted references to be cleared, got %+v", logs[0])
	}
}
