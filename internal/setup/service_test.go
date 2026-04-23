package setup

import (
	"context"
	"path/filepath"
	"testing"

	"main/internal/database"
	"main/internal/identity"
)

func TestCompleteOnlyRunsOnceAndPersistsOwner(t *testing.T) {
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

	identities := identity.NewService(container.DB())
	service := NewService(container.DB(), identities)

	owner, err := service.Complete(context.Background(), CompleteSetupInput{
		Username: "owner",
		Password: "owner1234",
	}, identity.ActionAuditContext{})
	if err != nil {
		t.Fatalf("failed to complete setup: %v", err)
	}
	if owner.Email != nil {
		t.Fatalf("expected setup owner email to be nil, got %q", *owner.Email)
	}
	if owner.Role != "owner" {
		t.Fatalf("expected owner role, got %+v", owner.Role)
	}

	_, err = service.Complete(context.Background(), CompleteSetupInput{
		Username: "other",
		Password: "password123",
	}, identity.ActionAuditContext{})
	if err != ErrAlreadyCompleted {
		t.Fatalf("expected ErrAlreadyCompleted, got %v", err)
	}

	state, err := service.GetState(context.Background())
	if err != nil {
		t.Fatalf("failed to read setup state: %v", err)
	}
	if !state.SetupCompleted {
		t.Fatal("expected setup state to be completed")
	}

	var ownerCount int
	if err := container.DB().QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'owner'`).Scan(&ownerCount); err != nil {
		t.Fatalf("failed to count owners: %v", err)
	}
	if ownerCount != 1 {
		t.Fatalf("expected exactly one owner, got %d", ownerCount)
	}
}
