package authorization

import "testing"

func TestViewerAuthorizationAppliesDynamicCapabilities(t *testing.T) {
	t.Parallel()

	service := NewService()

	viewer := service.ViewerAuthorization([]string{RoleAdmin}, ViewerOptions{
		AllowUserCreate: false,
		AllowSelfDelete: true,
	})

	if service.HasCapability(viewer.Capabilities, CapabilityManagementUsersCreate) {
		t.Fatal("expected management.users.create to be removed when create is disabled")
	}
	if !service.HasCapability(viewer.Capabilities, CapabilityAccountDeleteSelf) {
		t.Fatal("expected account.delete_self capability to be added when allowed")
	}
}

func TestCanManageUserEnforcesOwnerAdminBoundaries(t *testing.T) {
	t.Parallel()

	service := NewService()

	owner := UserTarget{UserID: 1, RoleKeys: []string{RoleOwner}, Status: "active"}
	admin := UserTarget{UserID: 2, RoleKeys: []string{RoleAdmin}, Status: "active"}
	user := UserTarget{UserID: 3, RoleKeys: []string{RoleUser}, Status: "active"}

	if service.CanManageUser(admin, owner, UserActionUpdate) {
		t.Fatal("expected admin to be unable to manage owner")
	}
	if service.CanManageUser(admin, admin, UserActionUpdate) {
		t.Fatal("expected admin to be unable to manage self")
	}
	if !service.CanManageUser(admin, user, UserActionDisable) {
		t.Fatal("expected admin to manage user")
	}
	if !service.CanManageUser(owner, admin, UserActionUpdate) {
		t.Fatal("expected owner to manage admin")
	}
}
