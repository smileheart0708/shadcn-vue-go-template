package httpapi

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"main/internal/auth"
	"main/internal/database"
	"main/internal/logging"
	"main/internal/users"
)

type testContext struct {
	handler   http.Handler
	store     *users.Store
	logStream *logging.Stream
	dataDir   string
	logs      *bytes.Buffer
	auth      auth.Options
	dbCleanup func()
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

func TestHealthz(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	output := ctx.logs.String()
	if !strings.Contains(output, "route=\"GET /api/healthz\"") {
		t.Fatalf("expected matched route in logs, got %q", output)
	}
	if !strings.Contains(output, "status=204") {
		t.Fatalf("expected status=204 in logs, got %q", output)
	}
	if !strings.Contains(output, "remote_addr=203.0.113.10") {
		t.Fatalf("expected stripped remote addr in logs, got %q", output)
	}
}

func TestAPIRoutesAreNotLoggedByDefault(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if output := strings.TrimSpace(ctx.logs.String()); output != "" {
		t.Fatalf("expected API logs to be disabled by default, got %q", output)
	}
}

func TestLoginAndMeFlow(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	token := loginAsBootstrapAdmin(t, ctx)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response testSuccessEnvelope[users.PublicUser]
	decodeJSONResponse(t, rec.Body.Bytes(), &response)

	if !response.Success {
		t.Fatal("expected success response")
	}
	if response.Data.Username != "admin" {
		t.Fatalf("expected username admin, got %q", response.Data.Username)
	}
	if response.Data.Role != users.RoleSuperAdmin {
		t.Fatalf("expected role %d, got %d", users.RoleSuperAdmin, response.Data.Role)
	}
	if !response.Data.MustChangePassword {
		t.Fatal("expected bootstrap admin to require password change")
	}
}

func TestLoginSetsRefreshCookieAndWritesLog(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	response, refreshCookie := loginBootstrapAdminSession(t, ctx)

	if refreshCookie == nil {
		t.Fatal("expected refresh cookie to be set")
	}
	if !refreshCookie.HttpOnly {
		t.Fatal("expected refresh cookie to be httpOnly")
	}
	if refreshCookie.Name != auth.ResolveRefreshCookieName(ctx.auth) {
		t.Fatalf("expected refresh cookie name %q, got %q", auth.ResolveRefreshCookieName(ctx.auth), refreshCookie.Name)
	}
	if refreshCookie.Path != auth.RefreshCookiePath {
		t.Fatalf("expected refresh cookie path %q, got %q", auth.RefreshCookiePath, refreshCookie.Path)
	}
	if response.AccessToken == "" {
		t.Fatal("expected access token in login response")
	}

	logs, err := ctx.store.ListAuthLogs(context.Background())
	if err != nil {
		t.Fatalf("failed to list auth logs: %v", err)
	}
	if len(logs) != 1 || logs[0].EventType != "login_success" || !logs[0].Success {
		t.Fatalf("expected one login_success log, got %+v", logs)
	}
}

func TestLoginSupportsCustomRefreshCookieName(t *testing.T) {
	t.Parallel()

	ctx := newTestContextWithAuthOptions(t, false, auth.Options{
		Issuer:             "test-suite",
		Secret:             []byte("test-secret"),
		TTL:                time.Hour,
		RefreshIdleTTL:     7 * 24 * time.Hour,
		RefreshAbsoluteTTL: 30 * 24 * time.Hour,
		RefreshCookieName:  "project_a_refresh_token",
	})
	_, refreshCookie := loginBootstrapAdminSession(t, ctx)

	if refreshCookie == nil {
		t.Fatal("expected refresh cookie to be set")
	}
	if refreshCookie.Name != "project_a_refresh_token" {
		t.Fatalf("expected refresh cookie name %q, got %q", "project_a_refresh_token", refreshCookie.Name)
	}
}

func TestLoginSupportsEmailIdentifier(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	password := "member123"
	createTestUser(t, ctx.store, users.CreateParams{
		Username:     "member",
		Email:        stringPointer("member@example.com"),
		PasswordHash: mustHashPassword(t, password),
		Role:         users.RoleUser,
	})

	loginBody := `{"identifier":"member@example.com","password":"member123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response testSuccessEnvelope[loginResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Data.User.Username != "member" {
		t.Fatalf("expected username member, got %q", response.Data.User.Username)
	}
}

func TestInvalidCredentialsReturnUnauthorized(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"identifier":"admin","password":"wrongpass"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	var response testErrorEnvelope
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Success {
		t.Fatal("expected unsuccessful response")
	}
	if response.Error.Code != "invalid_credentials" {
		t.Fatalf("expected invalid_credentials, got %q", response.Error.Code)
	}

	logs, err := ctx.store.ListAuthLogs(context.Background())
	if err != nil {
		t.Fatalf("failed to list auth logs: %v", err)
	}
	if len(logs) != 1 || logs[0].EventType != "login_failed" || logs[0].Success {
		t.Fatalf("expected one login_failed log, got %+v", logs)
	}
}

func TestRefreshRotatesTokenAndWritesLog(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	_, refreshCookie := loginBootstrapAdminSession(t, ctx)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(refreshCookie)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response testSuccessEnvelope[loginResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Data.AccessToken == "" {
		t.Fatal("expected refreshed access token")
	}

	nextCookie := findCookie(rec.Result().Cookies(), auth.ResolveRefreshCookieName(ctx.auth))
	if nextCookie == nil {
		t.Fatal("expected rotated refresh cookie")
	}
	if nextCookie.Value == refreshCookie.Value {
		t.Fatal("expected refresh token rotation to change cookie value")
	}

	logs, err := ctx.store.ListAuthLogs(context.Background())
	if err != nil {
		t.Fatalf("failed to list auth logs: %v", err)
	}
	if len(logs) != 2 || logs[1].EventType != "refresh_success" || !logs[1].Success {
		t.Fatalf("expected refresh_success log after login, got %+v", logs)
	}
}

func TestExpiredAccessTokenCanRefreshSession(t *testing.T) {
	t.Parallel()

	ctx := newTestContextWithAuthOptions(t, false, auth.Options{
		Issuer:             "test-suite",
		Secret:             []byte("test-secret"),
		TTL:                5 * time.Millisecond,
		RefreshIdleTTL:     7 * 24 * time.Hour,
		RefreshAbsoluteTTL: 30 * 24 * time.Hour,
	})
	response, refreshCookie := loginBootstrapAdminSession(t, ctx)

	time.Sleep(20 * time.Millisecond)

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+response.AccessToken)
	meRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected expired access token to fail with 401, got %d", meRec.Code)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(refreshRec, refreshReq)

	if refreshRec.Code != http.StatusOK {
		t.Fatalf("expected refresh status %d, got %d: %s", http.StatusOK, refreshRec.Code, refreshRec.Body.String())
	}
}

func TestPasswordChangeInvalidatesOldAccessAndRefreshTokens(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	loginResponse, refreshCookie := loginBootstrapAdminSession(t, ctx)
	bootstrapPassword := readBootstrapPassword(t, ctx.dataDir)

	changeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/account/password",
		strings.NewReader(`{"currentPassword":"`+bootstrapPassword+`","newPassword":"newsecure1"}`),
	)
	changeReq.Header.Set("Content-Type", "application/json")
	changeReq.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)
	changeReq.AddCookie(refreshCookie)
	changeRec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(changeRec, changeReq)

	if changeRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, changeRec.Code, changeRec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)
	meRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected old access token to be rejected after password change, got %d", meRec.Code)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(refreshRec, refreshReq)

	if refreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected old refresh token to be rejected after password change, got %d", refreshRec.Code)
	}

	logs, err := ctx.store.ListAuthLogs(context.Background())
	if err != nil {
		t.Fatalf("failed to list auth logs: %v", err)
	}
	if !containsAuthEvent(logs, "password_changed") || !containsAuthEvent(logs, "password_changed_forced_logout") {
		t.Fatalf("expected password change auth logs, got %+v", logs)
	}
}

func TestLogoutInvalidatesRefreshToken(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	_, refreshCookie := loginBootstrapAdminSession(t, ctx)

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.AddCookie(refreshCookie)
	logoutRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected logout status %d, got %d: %s", http.StatusOK, logoutRec.Code, logoutRec.Body.String())
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(refreshRec, refreshReq)

	if refreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected refresh token to be rejected after logout, got %d", refreshRec.Code)
	}
}

func TestProtectedRouteRejectsMissingToken(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	var response testErrorEnvelope
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Error.Code != "missing_token" {
		t.Fatalf("expected missing_token error, got %q", response.Error.Code)
	}
}

func TestUpdateProfileHandlesNullableEmail(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	token := loginAsBootstrapAdmin(t, ctx)

	req := httptest.NewRequest(http.MethodPatch, "/api/account/profile", strings.NewReader(`{"username":"admin-renamed","email":null}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response testSuccessEnvelope[users.PublicUser]
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Data.Username != "admin-renamed" {
		t.Fatalf("expected updated username, got %q", response.Data.Username)
	}
	if response.Data.Email != nil {
		t.Fatalf("expected nil email, got %v", *response.Data.Email)
	}
}

func TestUpdatePasswordClearsBootstrapFlagAndFile(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	token := loginAsBootstrapAdmin(t, ctx)
	bootstrapPassword := readBootstrapPassword(t, ctx.dataDir)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/account/password",
		strings.NewReader(`{"currentPassword":"`+bootstrapPassword+`","newPassword":"newsecure1"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response testSuccessEnvelope[users.PublicUser]
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Data.MustChangePassword {
		t.Fatal("expected bootstrap flag to be cleared")
	}

	if _, err := os.Stat(users.BootstrapPasswordPath(ctx.dataDir)); !os.IsNotExist(err) {
		t.Fatalf("expected bootstrap password file to be deleted, stat err=%v", err)
	}
}

func TestAvatarUploadStoresFileAndReturnsURL(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	token := loginAsBootstrapAdmin(t, ctx)

	body, contentType := newAvatarUploadRequest(t, "avatar.png", oneByOnePNG(t))
	req := httptest.NewRequest(http.MethodPost, "/api/account/avatar", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response testSuccessEnvelope[users.PublicUser]
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Data.AvatarURL == nil {
		t.Fatal("expected avatar URL in response")
	}

	avatarFiles, err := os.ReadDir(users.AvatarDir(ctx.dataDir))
	if err != nil {
		t.Fatalf("failed to read avatar dir: %v", err)
	}
	if len(avatarFiles) != 1 {
		t.Fatalf("expected 1 avatar file, got %d", len(avatarFiles))
	}
}

func TestAvatarUploadRejectsUnsupportedType(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	token := loginAsBootstrapAdmin(t, ctx)

	body, contentType := newAvatarUploadRequest(t, "avatar.txt", []byte("plain text"))
	req := httptest.NewRequest(http.MethodPost, "/api/account/avatar", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response testErrorEnvelope
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Error.Code != "avatar_invalid_type" {
		t.Fatalf("expected avatar_invalid_type, got %q", response.Error.Code)
	}
}

func TestDeleteAccountAllowsNormalUsers(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	user := createTestUser(t, ctx.store, users.CreateParams{
		Username:     "member",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleUser,
	})
	token := issueTokenForUser(t, ctx, user)

	req := httptest.NewRequest(http.MethodDelete, "/api/account", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if _, err := ctx.store.GetByID(context.Background(), user.ID); !errors.Is(err, users.ErrUserNotFound) {
		t.Fatalf("expected user to be deleted, got err=%v", err)
	}
}

func TestDeleteAccountRejectsSuperAdmin(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	token := loginAsBootstrapAdmin(t, ctx)

	req := httptest.NewRequest(http.MethodDelete, "/api/account", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	var response testErrorEnvelope
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	if response.Error.Code != "super_admin_delete_forbidden" {
		t.Fatalf("expected super_admin_delete_forbidden, got %q", response.Error.Code)
	}
}

func TestSPAFallbackServesIndexForClientRoute(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<title>SPA</title>") {
		t.Fatalf("expected SPA index response, got %q", rec.Body.String())
	}
}

func TestSPAFallbackPrefersGzipIndexWhenSupported(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set("Accept-Encoding", "gzip, br")
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("expected gzip content encoding, got %q", got)
	}
	if got := rec.Header().Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("expected Accept-Encoding in Vary header, got %q", got)
	}

	body := gunzipBody(t, rec.Body.Bytes())
	if !strings.Contains(body, "<title>SPA</title>") {
		t.Fatalf("expected gzipped index body, got %q", body)
	}
}

func TestUnknownAPIRouteDoesNotFallbackToSPA(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/missing", nil)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	if strings.Contains(rec.Body.String(), "<title>SPA</title>") {
		t.Fatalf("expected API 404 instead of SPA fallback, got %q", rec.Body.String())
	}
}

func TestAdminSystemLogsRequiresAuthentication(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/system-logs/stream", nil)
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAdminSystemLogsRequiresAdmin(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	user := createTestUser(t, ctx.store, users.CreateParams{
		Username:     "member",
		PasswordHash: mustHashPassword(t, "member123"),
		Role:         users.RoleUser,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/system-logs/stream", nil)
	req.Header.Set("Authorization", "Bearer "+issueTokenForUser(t, ctx, user))
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestAdminSystemLogsStreamsReplayAndLiveEntries(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	ctx.logStream.Publish(logging.StreamEntry{
		Timestamp: 1_700_000_001,
		Level:     "INFO",
		Message:   "boot complete",
		Text:      "boot complete",
		Source:    "app",
	})

	server := httptest.NewServer(ctx.handler)
	defer server.Close()

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, server.URL+"/api/admin/system-logs/stream?tail=1", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+loginAsBootstrapAdmin(t, ctx))

	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected event stream content type, got %q", contentType)
	}

	ctx.logStream.Publish(logging.StreamEntry{
		Timestamp: 1_700_000_002,
		Level:     "ERROR",
		Message:   "worker failed",
		Text:      "worker failed operation=refresh",
		Source:    "app",
	})

	events := readSystemLogEvents(t, response.Body, 2)
	cancel()

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Message != "boot complete" {
		t.Fatalf("expected replay event first, got %+v", events[0])
	}
	if events[1].Level != "ERROR" || !strings.Contains(events[1].Text, "operation=refresh") {
		t.Fatalf("expected live event second, got %+v", events[1])
	}
}

func TestAdminSystemLogsInvalidTailFallsBackAndLogs(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t, false)
	ctx.logStream.Publish(logging.StreamEntry{
		Timestamp: 1_700_000_100,
		Level:     "INFO",
		Message:   "seed",
		Text:      "seed",
		Source:    "app",
	})

	server := httptest.NewServer(ctx.handler)
	defer server.Close()

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, server.URL+"/api/admin/system-logs/stream?tail=abc", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+loginAsBootstrapAdmin(t, ctx))

	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer response.Body.Close()

	events := readSystemLogEvents(t, response.Body, 1)
	cancel()

	if len(events) != 1 || events[0].Message != "seed" {
		t.Fatalf("expected fallback replay entry, got %+v", events)
	}
	if !strings.Contains(ctx.logs.String(), "invalid system log tail parameter") {
		t.Fatalf("expected invalid tail warning in logs, got %q", ctx.logs.String())
	}
}

func newTestContext(t *testing.T, logAPIRequests bool) *testContext {
	return newTestContextWithAuthOptions(t, logAPIRequests, auth.Options{
		Issuer:             "test-suite",
		Secret:             []byte("test-secret"),
		TTL:                time.Hour,
		RefreshIdleTTL:     7 * 24 * time.Hour,
		RefreshAbsoluteTTL: 30 * 24 * time.Hour,
	})
}

func newTestContextWithAuthOptions(t *testing.T, logAPIRequests bool, authOptions auth.Options) *testContext {
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

	logs := &bytes.Buffer{}
	logger := testLogger(logs)
	store := users.NewStore(dbContainer.DB())
	if err := users.NewBootstrapManager(store, dataDir, logger).Ensure(context.Background()); err != nil {
		t.Fatalf("failed to bootstrap default super admin: %v", err)
	}
	logs.Reset()

	logStream := logging.NewStream(logging.StreamOptions{
		Capacity: logging.DefaultStreamCapacity,
	})

	handler := NewHandlerWithOptions(HandlerOptions{
		Logger:         logger,
		LogStream:      logStream,
		UserStore:      store,
		DataDir:        dataDir,
		FrontendFS:     os.DirFS(distDir),
		LogAPIRequests: logAPIRequests,
		Auth:           authOptions,
	})

	return &testContext{
		handler:   handler,
		store:     store,
		logStream: logStream,
		dataDir:   dataDir,
		logs:      logs,
		auth:      authOptions,
	}
}

func loginAsBootstrapAdmin(t *testing.T, ctx *testContext) string {
	t.Helper()

	response, _ := loginBootstrapAdminSession(t, ctx)
	return response.AccessToken
}

func loginBootstrapAdminSession(t *testing.T, ctx *testContext) (loginResponse, *http.Cookie) {
	t.Helper()

	password := readBootstrapPassword(t, ctx.dataDir)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"identifier":"admin","password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d: %s", rec.Code, rec.Body.String())
	}

	var response testSuccessEnvelope[loginResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &response)
	refreshCookie := findCookie(rec.Result().Cookies(), auth.ResolveRefreshCookieName(ctx.auth))
	return response.Data, refreshCookie
}

func issueTokenForUser(t *testing.T, ctx *testContext, user users.User) string {
	t.Helper()

	service := auth.NewService(ctx.auth, ctx.store)
	sessionID, _, err := service.CreateRefreshSession(context.Background(), user, "", "")
	if err != nil {
		t.Fatalf("failed to create refresh session: %v", err)
	}

	token, _, err := service.IssueToken(user, sessionID)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}
	return token
}

func readBootstrapPassword(t *testing.T, dataDir string) string {
	t.Helper()

	password, err := os.ReadFile(users.BootstrapPasswordPath(dataDir))
	if err != nil {
		t.Fatalf("failed to read bootstrap password: %v", err)
	}
	return strings.TrimSpace(string(password))
}

func createTestUser(t *testing.T, store *users.Store, params users.CreateParams) users.User {
	t.Helper()

	user, err := store.Create(context.Background(), params)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()

	passwordHash, err := users.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return passwordHash
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

func newAvatarUploadRequest(t *testing.T, fileName string, data []byte) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("avatar", fileName)
	if err != nil {
		t.Fatalf("failed to create multipart file: %v", err)
	}
	if _, err := fileWriter.Write(data); err != nil {
		t.Fatalf("failed to write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

func oneByOnePNG(t *testing.T) []byte {
	t.Helper()

	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO5qS3QAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("failed to decode PNG fixture: %v", err)
	}
	return data
}

func decodeJSONResponse(t *testing.T, payload []byte, dst any) {
	t.Helper()

	if err := json.Unmarshal(payload, dst); err != nil {
		t.Fatalf("failed to decode JSON response: %v; payload=%s", err, string(payload))
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

func testLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{}))
}

func readSystemLogEvents(t *testing.T, body io.Reader, count int) []logging.StreamEntry {
	t.Helper()

	reader := bufio.NewReader(body)
	events := make([]logging.StreamEntry, 0, count)
	deadline := time.Now().Add(2 * time.Second)

	for len(events) < count {
		if time.Now().After(deadline) {
			t.Fatalf("timed out reading system log events; received %d", len(events))
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("ReadString() error = %v", err)
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		var entry logging.StreamEntry
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data: "))), &entry); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		events = append(events, entry)
	}

	return events
}

func stringPointer(value string) *string {
	return &value
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func containsAuthEvent(records []users.AuthLogRecord, eventType string) bool {
	for _, record := range records {
		if record.EventType == eventType {
			return true
		}
	}
	return false
}
