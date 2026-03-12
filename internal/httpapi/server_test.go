package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	handler := NewHandler(testLogger(&logs))

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

func TestRequestLoggerFallsBackToURLPath(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	handler := NewHandler(testLogger(&logs))

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	output := logs.String()
	if !strings.Contains(output, "route=/missing") {
		t.Fatalf("expected fallback path in logs, got %q", output)
	}
	if !strings.Contains(output, "status=404") {
		t.Fatalf("expected status=404 in logs, got %q", output)
	}
}

func TestRequestLoggerDefaultsStatusToOK(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := testLogger(&logs)
	handler := requestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

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

func testLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{}))
}
