package httpapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"main/internal/audit"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/database"
	"main/internal/identity"
	"main/internal/logging"
	"main/internal/setup"
	"main/internal/systemsettings"
)

type testContext struct {
	handler         http.Handler
	authService     *auth.Service
	identityService *identity.Service
	settingsService *systemsettings.Service
	setupService    *setup.Service
	auditService    *audit.Service
	logStream       *logging.Stream
	dataDir         string
}

type testSuccessEnvelope[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

type testErrorEnvelope struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestInstallSetupOnlyRunsOnceAndCreatesOwnerSession(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)

	stateReq := httptest.NewRequest(http.MethodGet, "/api/install/state", nil)
	stateRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(stateRec, stateReq)

	if stateRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, stateRec.Code)
	}

	var stateEnvelope testSuccessEnvelope[installStateResponse]
	decodeJSONResponse(t, stateRec.Body.Bytes(), &stateEnvelope)
	if stateEnvelope.Data.SetupCompleted {
		t.Fatal("expected fresh install state to be pending")
	}

	session, refreshCookie := performSetup(t, ctx)
	if refreshCookie == nil {
		t.Fatal("expected refresh cookie after setup")
	}
	if got := session.Viewer.Authorization.RoleKeys; len(got) != 1 || got[0] != authorization.RoleOwner {
		t.Fatalf("expected owner role after setup, got %+v", got)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/install/setup", strings.NewReader(`{"username":"other","password":"password123"}`))
	secondReq.Header.Set("Content-Type", "application/json")
	secondRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, secondRec.Code, secondRec.Body.String())
	}

	decodeErrorCode(t, secondRec.Body.Bytes(), "setup_completed")
}

func TestSetupRequiredBlocksNormalAuthRoutesUntilInitialized(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"identifier":"owner","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	decodeErrorCode(t, rec.Body.Bytes(), "setup_required")
}

func TestSingleUserModeBlocksManagedUserCreationByDefault(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)

	req := httptest.NewRequest(http.MethodPost, "/api/management/users", strings.NewReader(`{"username":"member","password":"member123","roleKeys":["user"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
	decodeErrorCode(t, rec.Body.Bytes(), "forbidden")
}

func TestMultiUserRegisterRefreshReplayAndLogoutFlow(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)
	updateSystemSettings(t, ctx, ownerSession.AccessToken, `{"authMode":"multi_user","registrationMode":"public","adminUserCreateEnabled":true,"selfServiceAccountDeletionEnabled":true}`)

	_, memberCookie := registerUser(t, ctx, "member", "member@example.com", "member123")
	if memberCookie == nil {
		t.Fatal("expected refresh cookie for registered member")
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshReq.AddCookie(memberCookie)
	refreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(refreshRec, refreshReq)

	if refreshRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, refreshRec.Code, refreshRec.Body.String())
	}
	var refreshedEnvelope testSuccessEnvelope[sessionResponse]
	decodeJSONResponse(t, refreshRec.Body.Bytes(), &refreshedEnvelope)
	nextCookie := findCookie(refreshRec.Result().Cookies(), ctx.authServiceCookieName())
	if nextCookie == nil || nextCookie.Value == memberCookie.Value {
		t.Fatal("expected rotated refresh cookie")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+refreshedEnvelope.Data.AccessToken)
	meRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, meRec.Code)
	}

	replayReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	replayReq.AddCookie(memberCookie)
	replayRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(replayRec, replayReq)

	if replayRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, replayRec.Code)
	}
	decodeErrorCode(t, replayRec.Body.Bytes(), "invalid_refresh_token")

	postReplayRefreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	postReplayRefreshReq.AddCookie(nextCookie)
	postReplayRefreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(postReplayRefreshRec, postReplayRefreshReq)
	if postReplayRefreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, postReplayRefreshRec.Code)
	}

	_, latestSessionCookie := loginUser(t, ctx, "member", "member123")
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.AddCookie(latestSessionCookie)
	logoutRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, logoutRec.Code)
	}

	postLogoutRefreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	postLogoutRefreshReq.AddCookie(latestSessionCookie)
	postLogoutRefreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(postLogoutRefreshRec, postLogoutRefreshReq)
	if postLogoutRefreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, postLogoutRefreshRec.Code)
	}
}

func TestPasswordChangeInvalidatesOldAccessAndRefreshTokens(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)
	updateSystemSettings(t, ctx, ownerSession.AccessToken, `{"authMode":"multi_user","registrationMode":"public","adminUserCreateEnabled":true,"selfServiceAccountDeletionEnabled":true}`)
	memberSession, memberCookie := registerUser(t, ctx, "member", "member@example.com", "member123")

	changeReq := httptest.NewRequest(http.MethodPost, "/api/account/password", strings.NewReader(`{"currentPassword":"member123","newPassword":"member456"}`))
	changeReq.Header.Set("Content-Type", "application/json")
	changeReq.Header.Set("Authorization", "Bearer "+memberSession.AccessToken)
	changeReq.AddCookie(memberCookie)
	changeRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(changeRec, changeReq)

	if changeRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, changeRec.Code, changeRec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+memberSession.AccessToken)
	meRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, meRec.Code)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshReq.AddCookie(memberCookie)
	refreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, refreshRec.Code)
	}
}

func TestRoleChangeAndDisableInvalidateExistingSessions(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)
	updateSystemSettings(t, ctx, ownerSession.AccessToken, `{"authMode":"multi_user","registrationMode":"public","adminUserCreateEnabled":true,"selfServiceAccountDeletionEnabled":true}`)

	memberSession, memberCookie := registerUser(t, ctx, "member", "member@example.com", "member123")
	member := findUserByUsername(t, ctx, "member")

	updateReq := httptest.NewRequest(http.MethodPatch, "/api/management/users/"+strconv.FormatInt(member.ID, 10), strings.NewReader(`{"username":"member","email":"member@example.com","roleKeys":["admin"]}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	updateRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, updateRec.Code, updateRec.Body.String())
	}

	assertUnauthorizedMeAndRefresh(t, ctx, memberSession.AccessToken, memberCookie)

	adminSession, adminCookie := loginUser(t, ctx, "member", "member123")
	disableReq := httptest.NewRequest(http.MethodPost, "/api/management/users/"+strconv.FormatInt(member.ID, 10)+"/disable", nil)
	disableReq.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	disableRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(disableRec, disableReq)

	if disableRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, disableRec.Code, disableRec.Body.String())
	}

	assertUnauthorizedMeAndRefresh(t, ctx, adminSession.AccessToken, adminCookie)
}

func TestAdminCannotManageOwnerOrOtherAdminsAndOwnerRemainsVisible(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)
	updateSystemSettings(t, ctx, ownerSession.AccessToken, `{"authMode":"multi_user","registrationMode":"disabled","adminUserCreateEnabled":true,"selfServiceAccountDeletionEnabled":true}`)

	createManagedUser(t, ctx, ownerSession.AccessToken, `{"username":"admin-one","email":"admin-one@example.com","password":"admin1234","roleKeys":["admin"]}`)
	createManagedUser(t, ctx, ownerSession.AccessToken, `{"username":"member-one","email":"member-one@example.com","password":"member1234","roleKeys":["user"]}`)

	adminSession, _ := loginUser(t, ctx, "admin-one", "admin1234")
	listReq := httptest.NewRequest(http.MethodGet, "/api/management/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+adminSession.AccessToken)
	listRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, listRec.Code, listRec.Body.String())
	}

	var listEnvelope testSuccessEnvelope[managementUsersPageResponse]
	decodeJSONResponse(t, listRec.Body.Bytes(), &listEnvelope)

	var ownerActions []string
	var adminActions []string
	var memberActions []string
	for _, item := range listEnvelope.Data.Items {
		switch item.Username {
		case "owner":
			ownerActions = item.Actions
		case "admin-one":
			adminActions = item.Actions
		case "member-one":
			memberActions = item.Actions
		}
	}

	if len(ownerActions) != 0 {
		t.Fatalf("expected owner to be visible but not actionable for admin, got %+v", ownerActions)
	}
	if len(adminActions) != 0 {
		t.Fatalf("expected admin self row to be non-actionable, got %+v", adminActions)
	}
	if len(memberActions) == 0 {
		t.Fatalf("expected member row to remain actionable for admin, got %+v", memberActions)
	}

	owner := findUserByUsername(t, ctx, "owner")
	updateOwnerReq := httptest.NewRequest(http.MethodPatch, "/api/management/users/"+strconv.FormatInt(owner.ID, 10), strings.NewReader(`{"username":"owner","roleKeys":["user"]}`))
	updateOwnerReq.Header.Set("Content-Type", "application/json")
	updateOwnerReq.Header.Set("Authorization", "Bearer "+adminSession.AccessToken)
	updateOwnerRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(updateOwnerRec, updateOwnerReq)

	if updateOwnerRec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, updateOwnerRec.Code)
	}

	createAdminReq := httptest.NewRequest(http.MethodPost, "/api/management/users", strings.NewReader(`{"username":"admin-two","password":"admin1234","roleKeys":["admin"]}`))
	createAdminReq.Header.Set("Content-Type", "application/json")
	createAdminReq.Header.Set("Authorization", "Bearer "+adminSession.AccessToken)
	createAdminRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(createAdminRec, createAdminReq)

	if createAdminRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, createAdminRec.Code)
	}
	decodeErrorCode(t, createAdminRec.Body.Bytes(), "invalid_role_keys")
}

func TestManagedUserResponsesEncodeEmptySlicesAsJSONArrays(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)

	listReq := httptest.NewRequest(http.MethodGet, "/api/management/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	listRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"actions":[]`) {
		t.Fatalf("expected empty actions to encode as JSON array, got %s", listRec.Body.String())
	}

	updateSystemSettings(t, ctx, ownerSession.AccessToken, `{"authMode":"multi_user","registrationMode":"disabled","adminUserCreateEnabled":true,"selfServiceAccountDeletionEnabled":true}`)

	createReq := httptest.NewRequest(http.MethodPost, "/api/management/users", strings.NewReader(`{"username":"member","password":"member1234","roleKeys":["user"]}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	createRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, createRec.Code, createRec.Body.String())
	}
	if !strings.Contains(createRec.Body.String(), `"actions":[]`) {
		t.Fatalf("expected created user actions to encode as JSON array, got %s", createRec.Body.String())
	}
}

func TestAuditLogsEndpointIncludesSecurityEvents(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)

	req := httptest.NewRequest(http.MethodGet, "/api/management/audit-logs?page=1&pageSize=20", nil)
	req.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var envelope testSuccessEnvelope[audit.ListResult]
	decodeJSONResponse(t, rec.Body.Bytes(), &envelope)
	if len(envelope.Data.Items) == 0 {
		t.Fatal("expected setup audit log to be present")
	}
	if envelope.Data.Items[0].EventType != audit.EventSetupCompleted {
		t.Fatalf("expected first audit event %q, got %q", audit.EventSetupCompleted, envelope.Data.Items[0].EventType)
	}
}

func TestSPAFallbackAndGzipStillWork(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<title>SPA</title>") {
		t.Fatalf("expected SPA index response, got %q", rec.Body.String())
	}

	gzipReq := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	gzipReq.Header.Set("Accept-Encoding", "gzip")
	gzipRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(gzipRec, gzipReq)
	if gzipRec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip content encoding, got %q", gzipRec.Header().Get("Content-Encoding"))
	}
	if got := gunzipBody(t, gzipRec.Body.Bytes()); !strings.Contains(got, "<title>SPA</title>") {
		t.Fatalf("expected gzipped SPA body, got %q", got)
	}
}

func newTestContext(t *testing.T) *testContext {
	t.Helper()

	dataDir := t.TempDir()
	distDir := createTestDist(t)
	dbContainer, err := database.Open(context.Background(), database.Options{
		Path: filepath.Join(dataDir, "test.db"),
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := dbContainer.Close(); err != nil {
			t.Fatalf("failed to close test database: %v", err)
		}
	})

	if err := authorization.EnsureCatalog(context.Background(), dbContainer.DB()); err != nil {
		t.Fatalf("failed to seed authorization catalog: %v", err)
	}

	logStream := logging.NewStream(logging.StreamOptions{
		Capacity:  logging.DefaultStreamCapacity,
		Retention: 3650 * 24 * time.Hour,
	})

	authzService := authorization.NewService()
	identityService := identity.NewService(dbContainer.DB())
	settingsService := systemsettings.NewService(dbContainer.DB())
	setupService := setup.NewService(dbContainer.DB(), identityService)
	auditService := audit.NewService(dbContainer.DB())
	authService := auth.NewService(auth.Options{
		Issuer:             "test-suite",
		Secret:             []byte("test-secret"),
		TTL:                time.Hour,
		RefreshIdleTTL:     7 * 24 * time.Hour,
		RefreshAbsoluteTTL: 30 * 24 * time.Hour,
	}, dbContainer.DB(), identityService, authzService, settingsService)
	logger := logging.New()

	handler := NewHandlerWithOptions(HandlerOptions{
		Logger:         logger,
		LogStream:      logStream,
		Auth:           authService,
		Authorization:  authzService,
		Identity:       identityService,
		Setup:          setupService,
		SystemSettings: settingsService,
		Audit:          auditService,
		DataDir:        dataDir,
		FrontendFS:     os.DirFS(distDir),
		LogAPIRequests: false,
	})

	return &testContext{
		handler:         handler,
		authService:     authService,
		identityService: identityService,
		settingsService: settingsService,
		setupService:    setupService,
		auditService:    auditService,
		logStream:       logStream,
		dataDir:         dataDir,
	}
}

func (ctx *testContext) authServiceCookieName() string {
	return auth.ResolveRefreshCookieName(auth.Options{Secret: []byte("test-secret")})
}

func performSetup(t *testing.T, ctx *testContext) (sessionResponse, *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/install/setup", strings.NewReader(`{"username":"owner","password":"owner1234"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var envelope testSuccessEnvelope[sessionResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &envelope)
	return envelope.Data, findCookie(rec.Result().Cookies(), ctx.authServiceCookieName())
}

func updateSystemSettings(t *testing.T, ctx *testContext, token string, payload string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPatch, "/api/system/settings", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func registerUser(t *testing.T, ctx *testContext, username string, email string, password string) (sessionResponse, *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"`+username+`","email":"`+email+`","password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var envelope testSuccessEnvelope[sessionResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &envelope)
	return envelope.Data, findCookie(rec.Result().Cookies(), ctx.authServiceCookieName())
}

func loginUser(t *testing.T, ctx *testContext, identifier string, password string) (sessionResponse, *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"identifier":"`+identifier+`","password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var envelope testSuccessEnvelope[sessionResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &envelope)
	return envelope.Data, findCookie(rec.Result().Cookies(), ctx.authServiceCookieName())
}

func createManagedUser(t *testing.T, ctx *testContext, token string, payload string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/management/users", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
}

func findUserByUsername(t *testing.T, ctx *testContext, username string) identity.UserWithRoles {
	t.Helper()

	list, err := ctx.identityService.ListUsers(context.Background(), identity.ListUsersParams{Query: username, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	for _, item := range list.Items {
		if item.Username == username {
			return item
		}
	}
	t.Fatalf("user %q not found", username)
	return identity.UserWithRoles{}
}

func assertUnauthorizedMeAndRefresh(t *testing.T, ctx *testContext, accessToken string, refreshCookie *http.Cookie) {
	t.Helper()

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+accessToken)
	meRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, meRec.Code)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, refreshRec.Code)
	}
}

func createTestDist(t *testing.T) string {
	t.Helper()

	distDir := t.TempDir()
	assetsDir := filepath.Join(distDir, "assets")

	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("failed to create asset directory: %v", err)
	}

	writeTestFile(t, filepath.Join(distDir, "index.html"), "<!doctype html><html><head><title>SPA</title></head><body>index</body></html>")
	writeGzipFile(t, filepath.Join(distDir, "index.html.gz"), "<!doctype html><html><head><title>SPA</title></head><body>index</body></html>")
	writeTestFile(t, filepath.Join(assetsDir, "app.js"), "console.log('spa asset')\n")
	writeGzipFile(t, filepath.Join(assetsDir, "app.js.gz"), "console.log('spa asset')\n")

	return distDir
}

func writeTestFile(t *testing.T, filename string, content string) {
	t.Helper()
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", filename, err)
	}
}

func writeGzipFile(t *testing.T, filename string, content string) {
	t.Helper()

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatalf("failed to gzip %s: %v", filename, err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to finalize gzip %s: %v", filename, err)
	}
	if err := os.WriteFile(filename, compressed.Bytes(), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", filename, err)
	}
}

func decodeJSONResponse(t *testing.T, payload []byte, dst any) {
	t.Helper()
	if err := json.Unmarshal(payload, dst); err != nil {
		t.Fatalf("failed to decode JSON response: %v; payload=%s", err, string(payload))
	}
}

func decodeErrorCode(t *testing.T, payload []byte, wantCode string) {
	t.Helper()

	var envelope testErrorEnvelope
	decodeJSONResponse(t, payload, &envelope)
	if envelope.Error.Code != wantCode {
		t.Fatalf("expected error code %q, got %q", wantCode, envelope.Error.Code)
	}
}

func gunzipBody(t *testing.T, body []byte) string {
	t.Helper()

	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to decode gzip body: %v", err)
	}
	return string(decoded)
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
