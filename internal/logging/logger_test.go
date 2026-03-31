package logging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestStreamPublishAndSnapshot(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{Capacity: 3})
	base := time.Now().UTC()

	for index := range 5 {
		stream.Publish(StreamEntry{
			Timestamp: base.Add(time.Duration(index) * time.Second).Unix(),
			Level:     "INFO",
			Message:   "entry",
			Text:      "entry",
			Source:    "app",
		})
	}

	snapshot := stream.Snapshot(10)
	if len(snapshot) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(snapshot))
	}
	if !(snapshot[0].Timestamp < snapshot[1].Timestamp && snapshot[1].Timestamp < snapshot[2].Timestamp) {
		t.Fatalf("expected timestamps to increase, got %+v", snapshot)
	}
	if snapshot[0].ID <= 0 || snapshot[2].ID <= snapshot[0].ID {
		t.Fatalf("expected ascending ids, got %+v", snapshot)
	}
}

func TestStreamPrunesExpiredEntriesByRetention(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{Capacity: 4, Retention: time.Hour})
	now := time.Now().UTC()
	recentTimestamp := now.Add(-30 * time.Minute).Unix()
	stream.entries = []StreamEntry{
		{
			ID:        1,
			Timestamp: now.Add(-2 * time.Hour).Unix(),
			Level:     "INFO",
			Message:   "expired",
			Text:      "expired",
			Source:    "app",
		},
		{
			ID:        2,
			Timestamp: recentTimestamp,
			Level:     "INFO",
			Message:   "recent",
			Text:      "recent",
			Source:    "app",
		},
	}

	snapshot := stream.Snapshot(10)
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 retained entry, got %d", len(snapshot))
	}
	if snapshot[0].Timestamp != recentTimestamp {
		t.Fatalf("expected retained entry to be the recent log, got %+v", snapshot[0])
	}
}

func TestStreamDropsSlowSubscriber(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{Capacity: 4})
	subscription := stream.Subscribe()

	for index := range defaultSubscriberSize + 1 {
		stream.Publish(StreamEntry{
			Timestamp: int64(index + 1),
			Level:     "INFO",
			Message:   "entry",
			Text:      "entry",
			Source:    "app",
		})
	}

	if reason := subscription.Reason(); reason != "slow_consumer" {
		t.Fatalf("expected slow_consumer reason, got %q", reason)
	}
	if count := stream.SubscriberCount(); count != 0 {
		t.Fatalf("expected no subscribers, got %d", count)
	}
}

func TestStreamHandlerSanitizesAndRendersEntry(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{Capacity: 10})
	handler := NewStreamHandler(stream, "app")
	logger := slog.New(handler)

	logger.InfoContext(
		context.Background(),
		"line one\nline two",
		"request_id", "req-123",
		"log_source", "http_access",
		"secret", "should-hide",
		"payload", map[string]any{"status": 200},
	)

	entries := stream.Snapshot(1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Source != "http_access" {
		t.Fatalf("expected source override, got %q", entry.Source)
	}
	if strings.Contains(entry.Text, "should-hide") {
		t.Fatalf("expected sensitive value to be hidden, got %q", entry.Text)
	}
	if strings.Contains(entry.Text, "\n") {
		t.Fatalf("expected single line text, got %q", entry.Text)
	}
	if !strings.Contains(entry.Text, "request_id=req-123") {
		t.Fatalf("expected request id in text, got %q", entry.Text)
	}
}

func TestStreamHandlerRendersErrorValuesAsMessages(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{Capacity: 10})
	logger := slog.New(NewStreamHandler(stream, "app"))

	logger.ErrorContext(
		context.Background(),
		"failed to write auth log",
		"error", fmt.Errorf("users: failed to insert auth log: %w", errors.New("constraint failed: FOREIGN KEY constraint failed (787)")),
		"event_type", "refresh_failed",
	)

	entries := stream.Snapshot(1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if strings.Contains(entry.Text, "error={}") {
		t.Fatalf("expected error details instead of empty object, got %q", entry.Text)
	}
	if !strings.Contains(entry.Text, `error="users: failed to insert auth log: constraint failed: FOREIGN KEY constraint failed (787)"`) {
		t.Fatalf("expected rendered error message in text, got %q", entry.Text)
	}
}

func TestLogStartupBanner(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{}))

	LogStartupBanner(logger, ":8080", "/tmp/data")

	output := buf.String()
	if strings.Count(output, bannerDivider) != 2 {
		t.Fatalf("expected divider twice, got %q", output)
	}
	if !strings.Contains(output, "listen: http://localhost:8080") {
		t.Fatalf("expected listen line in banner, got %q", output)
	}
	if !strings.Contains(output, "data_dir: /tmp/data") {
		t.Fatalf("expected data_dir line in banner, got %q", output)
	}
}
