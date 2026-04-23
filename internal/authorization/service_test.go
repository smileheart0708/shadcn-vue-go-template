package authorization

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestViewerAuthorizationAppliesDynamicCapabilities(t *testing.T) {
	t.Parallel()

	service := NewService()

	viewer := service.ViewerAuthorization(RoleUser, ViewerOptions{
		AllowSelfDelete: true,
	})

	if service.HasCapability(viewer.Capabilities, CapabilityManagementUsersCreate) {
		t.Fatal("expected regular users to never receive management.users.create")
	}
	if !service.HasCapability(viewer.Capabilities, CapabilityAccountDeleteSelf) {
		t.Fatal("expected account.delete_self capability to be added when allowed")
	}
}

func TestViewerAuthorizationAlwaysEncodesEmptyCapabilitiesAsArray(t *testing.T) {
	t.Parallel()

	service := NewService()

	viewer := service.ViewerAuthorization(RoleUser, ViewerOptions{})
	if viewer.Capabilities == nil {
		t.Fatal("expected empty capabilities slice to be non-nil")
	}

	payload, err := json.Marshal(viewer)
	if err != nil {
		t.Fatalf("failed to marshal viewer authorization: %v", err)
	}
	if !strings.Contains(string(payload), `"capabilities":[]`) {
		t.Fatalf("expected capabilities to encode as JSON array, got %s", string(payload))
	}
}

func TestCanManageUserEnforcesOwnerBoundaries(t *testing.T) {
	t.Parallel()

	service := NewService()

	owner := UserTarget{UserID: 1, Role: RoleOwner, Status: "active"}
	user := UserTarget{UserID: 2, Role: RoleUser, Status: "active"}
	otherOwner := UserTarget{UserID: 3, Role: RoleOwner, Status: "active"}

	if service.CanManageUser(user, owner, UserActionUpdate) {
		t.Fatal("expected regular users to be unable to manage owner")
	}
	if service.CanManageUser(owner, owner, UserActionUpdate) {
		t.Fatal("expected owner to be unable to manage self")
	}
	if !service.CanManageUser(owner, user, UserActionDisable) {
		t.Fatal("expected owner to manage regular user")
	}
	if service.CanManageUser(owner, otherOwner, UserActionUpdate) {
		t.Fatal("expected owner to be unable to manage another owner")
	}
}
