package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLogStartupBanner(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{}))

	LogStartupBanner(logger, ":8080", "/tmp/data")

	output := buf.String()
	if strings.Count(output, bannerDivider) != 2 {
		t.Fatalf("expected divider twice, got %q", output)
	}
	if !strings.Contains(output, "listen: :8080") {
		t.Fatalf("expected listen line in banner, got %q", output)
	}
	if !strings.Contains(output, "data_dir: /tmp/data") {
		t.Fatalf("expected data_dir line in banner, got %q", output)
	}
}
