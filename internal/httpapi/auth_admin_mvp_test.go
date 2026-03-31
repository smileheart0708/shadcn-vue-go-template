package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"main/internal/auth"
	"main/internal/systemsettings"
	"main/internal/users"
)

type testRegistrationPolicy struct {
	RegistrationMode string `json:"registrationMode"`
}

type testAdminUsersPage struct {
	Items    []users.AdminUser `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Total    int               `json:"total"`
}

func TestRegistrationPolicyDefaultsAndCanBeUpdated(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/registration-policy", nil)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var policyResponse testSuccessEnvelope[testRegistrationPolicy]
	decodeJSONResponse(t, rec.Body.Bytes(), &policyResponse)
	if policyResponse.Data.RegistrationMode != string(systemsettings.RegistrationModeDisabled) {
		t.Fatalf("expected default registration mode %q, got %q", systemsettings.RegistrationModeDisabled, policyResponse.Data.RegistrationMode)
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/api/admin/system-settings/registration", strings.NewReader(`{"registrationMode":"disabled"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+loginAsBootstrapAdmin(t, ctx))
	updateRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, updateRec.Code, updateRec.Body.String())
	}

	rec = httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	decodeJSONResponse(t, rec.Body.Bytes(), &policyResponse)
	if policyResponse.Data.RegistrationMode != string(systemsettings.RegistrationModeDisabled) {
		t.Fatalf("expected updated registration mode %q, got %q", systemsettings.RegistrationModeDisabled, policyResponse.Data.RegistrationMode)
	}
}

func TestRegisterCreatesUserAndAuthenticatesSession(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	settingsStore := systemsettings.NewStore(ctx.store.DB())
	if _, err := settingsStore.UpdateRegistrationMode(context.Background(), systemsettings.RegistrationModePassword); err != nil {
		t.Fatalf("failed to enable registration: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"member","email":"member@example.com","password":"member123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response testSuccessEnvelope[loginResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Data.User.Username != "member" {
		t.Fatalf("expected username member, got %q", response.Data.User.Username)
	}
	if response.Data.User.Role != users.RoleUser {
		t.Fatalf("expected role %d, got %d", users.RoleUser, response.Data.User.Role)
	}

	refreshCookie := findCookie(rec.Result().Cookies(), auth.ResolveRefreshCookieName(ctx.auth))
	if refreshCookie == nil {
		t.Fatal("expected refresh cookie after registration")
	}

	user, err := ctx.store.FindByIdentifier(context.Background(), "member@example.com")
	if err != nil {
		t.Fatalf("failed to load registered user: %v", err)
	}
	if user.IsBanned {
		t.Fatal("expected registered user to be active")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+response.Data.AccessToken)
	meRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, meRec.Code, meRec.Body.String())
	}
}

func TestRegisterRejectsWhenDisabled(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	settingsStore := systemsettings.NewStore(ctx.store.DB())
	if _, err := settingsStore.UpdateRegistrationMode(context.Background(), systemsettings.RegistrationModeDisabled); err != nil {
		t.Fatalf("failed to disable registration: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"member","password":"member123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}

	var response testErrorEnvelope
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Error.Code != "registration_disabled" {
		t.Fatalf("expected registration_disabled, got %q", response.Error.Code)
	}
}

func TestRegisterRejectsDuplicateUsername(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	settingsStore := systemsettings.NewStore(ctx.store.DB())
	if _, err := settingsStore.UpdateRegistrationMode(context.Background(), systemsettings.RegistrationModePassword); err != nil {
		t.Fatalf("failed to enable registration: %v", err)
	}
	createTestUser(t, ctx.store, users.CreateParams{
		Username:     "member",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleUser,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"member","password":"member123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rec.Code, rec.Body.String())
	}

	var response testErrorEnvelope
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Error.Code != "username_taken" {
		t.Fatalf("expected username_taken, got %q", response.Error.Code)
	}
}

func TestBannedUserCannotLogin(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	user := createTestUser(t, ctx.store, users.CreateParams{
		Username:     "member",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleUser,
	})
	if _, err := ctx.store.Ban(context.Background(), user.ID); err != nil {
		t.Fatalf("failed to ban user: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"identifier":"member","password":"member123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}

	var response testErrorEnvelope
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Error.Code != "account_banned" {
		t.Fatalf("expected account_banned, got %q", response.Error.Code)
	}
}

func TestAdminBanInvalidatesExistingSessions(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	createTestUser(t, ctx.store, users.CreateParams{
		Username:     "member",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleUser,
	})

	memberLogin, memberRefreshCookie := loginWithCredentials(t, ctx, "member", "member123")
	member, err := ctx.store.FindByIdentifier(context.Background(), "member")
	if err != nil {
		t.Fatalf("failed to load member: %v", err)
	}

	banReq := httptest.NewRequest(http.MethodPost, "/api/admin/users/"+strconvFormatInt(member.ID)+"/ban", nil)
	banReq.Header.Set("Authorization", "Bearer "+loginAsBootstrapAdmin(t, ctx))
	banRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(banRec, banReq)

	if banRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, banRec.Code, banRec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+memberLogin.AccessToken)
	meRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, meRec.Code)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshReq.AddCookie(memberRefreshCookie)
	refreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(refreshRec, refreshReq)

	if refreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, refreshRec.Code)
	}
}

func TestAdminUsersListSupportsFiltersAndPagination(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	createTestUser(t, ctx.store, users.CreateParams{
		Username:     "alpha-user",
		Email:        new("alpha@example.com"),
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleUser,
	})
	bannedUser := createTestUser(t, ctx.store, users.CreateParams{
		Username:     "banned-user",
		Email:        new("banned@example.com"),
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleUser,
	})
	if _, err := ctx.store.Ban(context.Background(), bannedUser.ID); err != nil {
		t.Fatalf("failed to ban user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users?status=banned&page=1&pageSize=1", nil)
	req.Header.Set("Authorization", "Bearer "+loginAsBootstrapAdmin(t, ctx))
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response testSuccessEnvelope[testAdminUsersPage]
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Data.Total != 1 {
		t.Fatalf("expected total 1, got %d", response.Data.Total)
	}
	if response.Data.Page != 1 || response.Data.PageSize != 1 {
		t.Fatalf("unexpected pagination payload: %+v", response.Data)
	}
	if len(response.Data.Items) != 1 || response.Data.Items[0].Status != users.UserStatusBanned {
		t.Fatalf("expected one banned user, got %+v", response.Data.Items)
	}

	searchReq := httptest.NewRequest(http.MethodGet, "/api/admin/users?q=alpha&role=0", nil)
	searchReq.Header.Set("Authorization", "Bearer "+loginAsBootstrapAdmin(t, ctx))
	searchRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, searchRec.Code, searchRec.Body.String())
	}

	decodeJSONResponse(t, searchRec.Body.Bytes(), &response)
	if response.Data.Total != 1 || len(response.Data.Items) != 1 || response.Data.Items[0].Username != "alpha-user" {
		t.Fatalf("expected alpha-user search result, got %+v", response.Data)
	}
}

func TestAdminCannotCreateOrEditAdminAccounts(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	adminActor := createTestUser(t, ctx.store, users.CreateParams{
		Username:     "admin-actor",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleAdmin,
	})
	adminTarget := createTestUser(t, ctx.store, users.CreateParams{
		Username:     "admin-target",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleAdmin,
	})
	adminToken := issueTokenForUser(t, ctx, adminActor)

	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(`{"username":"bad-admin","password":"member123","role":1}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+adminToken)
	createRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, createRec.Code, createRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/api/admin/users/"+strconvFormatInt(adminTarget.ID), strings.NewReader(`{"username":"admin-target","role":0}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+adminToken)
	updateRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, updateRec.Code, updateRec.Body.String())
	}
}

func TestAdminUserManagementRejectsSelfBan(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	adminUser := createTestUser(t, ctx.store, users.CreateParams{
		Username:     "admin-actor",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleAdmin,
	})
	adminToken := issueTokenForUser(t, ctx, adminUser)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/users/"+strconvFormatInt(adminUser.ID)+"/ban", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

func loginWithCredentials(t *testing.T, ctx *testContext, identifier string, password string) (loginResponse, *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"identifier":"`+identifier+`","password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d: %s", rec.Code, rec.Body.String())
	}

	var response testSuccessEnvelope[loginResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	refreshCookie := findCookie(rec.Result().Cookies(), auth.ResolveRefreshCookieName(ctx.auth))
	if refreshCookie == nil {
		t.Fatal("expected refresh cookie")
	}
	return response.Data, refreshCookie
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
