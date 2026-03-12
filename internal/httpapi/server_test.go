package httpapi

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHealthz(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	handler := newTestHandler(t, &logs)

	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	output := logs.String()
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

func TestLoginAndMeFlow(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, nil)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"admin123456"}`),
	)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()

	handler.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, loginRec.Code)
	}

	var loginPayload loginResponse
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	if loginPayload.AccessToken == "" {
		t.Fatal("expected access token in login response")
	}
	if loginPayload.User.Email != "admin@example.com" {
		t.Fatalf("expected email admin@example.com, got %q", loginPayload.User.Email)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+loginPayload.AccessToken)
	meRec := httptest.NewRecorder()

	handler.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, meRec.Code)
	}

	var mePayload meResponse
	if err := json.Unmarshal(meRec.Body.Bytes(), &mePayload); err != nil {
		t.Fatalf("failed to decode me response: %v", err)
	}

	if mePayload.User.Name != "Administrator" {
		t.Fatalf("expected user name Administrator, got %q", mePayload.User.Name)
	}
}

func TestProtectedRouteRejectsMissingToken(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), `"code":"missing_token"`) {
		t.Fatalf("expected missing token error body, got %q", rec.Body.String())
	}
}

func TestSPAFallbackServesIndexForClientRoute(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "<title>SPA</title>") {
		t.Fatalf("expected SPA index response, got %q", rec.Body.String())
	}
}

func TestSPAFallbackPrefersGzipIndexWhenSupported(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set("Accept-Encoding", "gzip, br")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

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

func TestStaticAssetsPreferGzipWhenSupported(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("expected gzip content encoding, got %q", got)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "javascript") {
		t.Fatalf("expected javascript content type, got %q", got)
	}

	body := gunzipBody(t, rec.Body.Bytes())
	if body != "console.log('spa asset')\n" {
		t.Fatalf("unexpected asset body %q", body)
	}
}

func TestMissingStaticAssetReturnsNotFound(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestUnknownAPIRouteDoesNotFallbackToSPA(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/missing", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	if strings.Contains(rec.Body.String(), "<title>SPA</title>") {
		t.Fatalf("expected API 404 instead of SPA fallback, got %q", rec.Body.String())
	}
}

func TestRequestLoggerFallsBackToURLPathForSPARoutes(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	handler := newTestHandler(t, &logs)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	output := logs.String()
	if !strings.Contains(output, "route=/dashboard") {
		t.Fatalf("expected fallback path in logs, got %q", output)
	}
	if !strings.Contains(output, "status=200") {
		t.Fatalf("expected status=200 in logs, got %q", output)
	}
}

func TestRequestLoggerDefaultsStatusToOK(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := testLogger(&logs)
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}), requestLogger(logger))

	req := httptest.NewRequest(http.MethodGet, "/implicit-ok", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	output := logs.String()
	if !strings.Contains(output, "route=/implicit-ok") {
		t.Fatalf("expected route fallback in logs, got %q", output)
	}
	if !strings.Contains(output, "status=200") {
		t.Fatalf("expected status=200 in logs, got %q", output)
	}
}

func newTestHandler(t *testing.T, logs *bytes.Buffer) http.Handler {
	t.Helper()

	distDir := createTestDist(t)
	logger := slog.Default()
	if logs != nil {
		logger = testLogger(logs)
	}

	return NewHandlerWithOptions(HandlerOptions{
		Logger:     logger,
		FrontendFS: os.DirFS(distDir),
		Auth: AuthOptions{
			Issuer: "test-suite",
			Secret: []byte("test-secret"),
			TTL:    time.Hour,
			User: AuthUser{
				Email: "admin@example.com",
				Name:  "Administrator",
			},
			Password: "admin123456",
		},
	})
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
