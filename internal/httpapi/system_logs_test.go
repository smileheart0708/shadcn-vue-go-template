package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseSystemLogTail(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "missing replays full buffer", want: 0},
		{name: "all replays full buffer", raw: "all", want: 0},
		{name: "uppercase all replays full buffer", raw: "ALL", want: 0},
		{name: "latest 100", raw: "100", want: 100},
		{name: "latest 200", raw: "200", want: 200},
		{name: "latest 500", raw: "500", want: 500},
		{name: "latest 1000", raw: "1000", want: 1000},
		{name: "invalid falls back to full buffer", raw: "invalid", want: 0},
		{name: "unsupported number falls back to full buffer", raw: "50", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/management/system-logs/stream", nil)
			if tt.raw != "" {
				query := req.URL.Query()
				query.Set("tail", tt.raw)
				req.URL.RawQuery = query.Encode()
			}

			if got := parseSystemLogTail(req, logger); got != tt.want {
				t.Fatalf("expected tail %d, got %d", tt.want, got)
			}
		})
	}
}
