package setup

import (
	"context"
	"path/filepath"
	"testing"

	"main/internal/authorization"
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

	if err := authorization.EnsureCatalog(context.Background(), container.DB()); err != nil {
		t.Fatalf("failed to seed authorization catalog: %v", err)
	}

	identities := identity.NewService(container.DB())
	service := NewService(container.DB(), identities)

	owner, err := service.Complete(context.Background(), CompleteSetupInput{
		Username: "owner",
		Email:    new("owner@example.com"),
		Password: "owner1234",
	}, identity.ActionAuditContext{})
	if err != nil {
		t.Fatalf("failed to complete setup: %v", err)
	}
	if got := owner.RoleKeys; len(got) != 1 || got[0] != authorization.RoleOwner {
		t.Fatalf("expected owner role, got %+v", got)
	}

	_, err = service.Complete(context.Background(), CompleteSetupInput{
		Username: "other",
		Email:    new("other@example.com"),
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
	if err := container.DB().QueryRow(`SELECT COUNT(*) FROM user_roles WHERE role_key = ?`, authorization.RoleOwner).Scan(&ownerCount); err != nil {
		t.Fatalf("failed to count owners: %v", err)
	}
	if ownerCount != 1 {
		t.Fatalf("expected exactly one owner, got %d", ownerCount)
	}
}
