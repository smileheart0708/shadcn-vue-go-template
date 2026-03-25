package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const (
	DefaultReplayLimit    = 200
	DefaultStreamCapacity = 1000
	defaultSubscriberSize = 128
)

type StreamEntry struct {
	ID        int64  `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Text      string `json:"text"`
	Source    string `json:"source"`
}

type StreamOptions struct {
	Capacity int
}

type Stream struct {
	mu          sync.RWMutex
	nextEntryID int64
	nextSubID   int64
	capacity    int
	entries     []StreamEntry
	subscribers map[int64]*StreamSubscription
}

type StreamSubscription struct {
	id      int64
	stream  *Stream
	entries chan StreamEntry

	mu     sync.RWMutex
	reason string
	closed bool
}

func NewStream(options StreamOptions) *Stream {
	capacity := options.Capacity
	if capacity <= 0 {
		capacity = DefaultStreamCapacity
	}

	return &Stream{
		capacity:    capacity,
		entries:     make([]StreamEntry, 0, capacity),
		subscribers: make(map[int64]*StreamSubscription),
	}
}

func (s *Stream) Publish(entry StreamEntry) StreamEntry {
	if s == nil {
		return entry
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextEntryID++
	entry.ID = s.nextEntryID
	if entry.Timestamp <= 0 {
		entry.Timestamp = time.Now().Unix()
	}
	if entry.Level == "" {
		entry.Level = strings.ToUpper(slog.LevelInfo.String())
	}
	if entry.Source == "" {
		entry.Source = "app"
	}

	s.entries = append(s.entries, entry)
	if len(s.entries) > s.capacity {
		s.entries = append([]StreamEntry(nil), s.entries[len(s.entries)-s.capacity:]...)
	}

	for id, subscription := range s.subscribers {
		select {
		case subscription.entries <- entry:
		default:
			subscription.closeLocked("slow_consumer")
			delete(s.subscribers, id)
		}
	}

	return entry
}

func (s *Stream) Snapshot(tail int) []StreamEntry {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if tail <= 0 || tail > len(s.entries) {
		tail = len(s.entries)
	}

	start := len(s.entries) - tail
	snapshot := make([]StreamEntry, tail)
	copy(snapshot, s.entries[start:])
	return snapshot
}

func (s *Stream) Subscribe() *StreamSubscription {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSubID++
	subscription := &StreamSubscription{
		id:      s.nextSubID,
		stream:  s,
		entries: make(chan StreamEntry, defaultSubscriberSize),
	}
	s.subscribers[subscription.id] = subscription
	return subscription
}

func (s *Stream) Unsubscribe(subscription *StreamSubscription, reason string) {
	if s == nil || subscription == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.subscribers[subscription.id]
	if !ok || current != subscription {
		return
	}

	subscription.closeLocked(reason)
	delete(s.subscribers, subscription.id)
}

func (s *Stream) SubscriberCount() int {
	if s == nil {
		return 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers)
}

func (s *StreamSubscription) Entries() <-chan StreamEntry {
	if s == nil {
		return nil
	}

	return s.entries
}

func (s *StreamSubscription) Close(reason string) {
	if s == nil || s.stream == nil {
		return
	}

	s.stream.Unsubscribe(s, reason)
}

func (s *StreamSubscription) Reason() string {
	if s == nil {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reason
}

func (s *StreamSubscription) closeLocked(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		if s.reason == "" && reason != "" {
			s.reason = reason
		}
		return
	}

	s.closed = true
	s.reason = reason
	close(s.entries)
}

type streamHandler struct {
	stream      *Stream
	defaultSrc  string
	prefixAttrs []scopedAttr
	groups      []string
}

type scopedAttr struct {
	groups []string
	attr   slog.Attr
}

type renderedAttr struct {
	key         string
	value       string
	sourceValue string
	hidden      bool
}

func NewStreamHandler(stream *Stream, source string) slog.Handler {
	if source == "" {
		source = "app"
	}

	return &streamHandler{
		stream:     stream,
		defaultSrc: source,
	}
}

func (h *streamHandler) Enabled(context.Context, slog.Level) bool {
	return h.stream != nil
}

func (h *streamHandler) Handle(_ context.Context, record slog.Record) error {
	if h == nil || h.stream == nil {
		return nil
	}

	rendered := make([]renderedAttr, 0, len(h.prefixAttrs)+8)
	for _, attr := range h.prefixAttrs {
		rendered = appendRenderedAttrs(rendered, attr.groups, attr.attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		rendered = appendRenderedAttrs(rendered, h.groups, attr)
		return true
	})

	source := h.defaultSrc
	parts := make([]string, 0, len(rendered)+1)
	message := sanitizeLogText(record.Message)
	if message != "" {
		parts = append(parts, message)
	}

	for _, attr := range rendered {
		if attr.sourceValue != "" {
			source = attr.sourceValue
		}
		if attr.hidden || attr.key == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", attr.key, attr.value))
	}

	text := strings.TrimSpace(strings.Join(parts, " "))
	if text == "" {
		text = message
	}

	h.stream.Publish(StreamEntry{
		Timestamp: record.Time.Unix(),
		Level:     strings.ToUpper(record.Level.String()),
		Message:   message,
		Text:      sanitizeLogText(text),
		Source:    source,
	})
	return nil
}

func (h *streamHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	clone := h.clone()
	for _, attr := range attrs {
		clone.prefixAttrs = append(clone.prefixAttrs, scopedAttr{
			groups: append([]string(nil), h.groups...),
			attr:   attr,
		})
	}
	return clone
}

func (h *streamHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	clone := h.clone()
	clone.groups = append(append([]string(nil), h.groups...), name)
	return clone
}

func (h *streamHandler) clone() *streamHandler {
	if h == nil {
		return &streamHandler{}
	}

	return &streamHandler{
		stream:      h.stream,
		defaultSrc:  h.defaultSrc,
		prefixAttrs: append([]scopedAttr(nil), h.prefixAttrs...),
		groups:      append([]string(nil), h.groups...),
	}
}

func appendRenderedAttrs(items []renderedAttr, groups []string, attr slog.Attr) []renderedAttr {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return items
	}

	if attr.Value.Kind() == slog.KindGroup {
		nextGroups := groups
		if attr.Key != "" {
			nextGroups = append(append([]string(nil), groups...), attr.Key)
		}
		for _, item := range attr.Value.Group() {
			items = appendRenderedAttrs(items, nextGroups, item)
		}
		return items
	}

	key := attr.Key
	if len(groups) > 0 {
		key = strings.Join(append(append([]string(nil), groups...), key), ".")
	}

	if hiddenLogKey(key) {
		return items
	}

	value := sanitizeLogText(renderSlogValue(attr.Value))
	rendered := renderedAttr{
		key:   key,
		value: value,
	}
	switch key {
	case "log_source", "source":
		rendered.sourceValue = value
		rendered.hidden = true
	}
	return append(items, rendered)
}

func renderSlogValue(value slog.Value) string {
	switch value.Kind() {
	case slog.KindString:
		return value.String()
	case slog.KindBool:
		if value.Bool() {
			return "true"
		}
		return "false"
	case slog.KindInt64:
		return fmt.Sprintf("%d", value.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%d", value.Uint64())
	case slog.KindFloat64:
		return fmt.Sprintf("%v", value.Float64())
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindAny:
		fallthrough
	default:
		raw := value.Any()
		data, err := json.Marshal(raw)
		if err != nil {
			return fmt.Sprintf("%v", raw)
		}
		return string(data)
	}
}

func hiddenLogKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if lower == "" {
		return true
	}

	sensitiveParts := []string{"token", "secret", "password", "authorization", "cookie"}
	for _, part := range sensitiveParts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

func sanitizeLogText(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.Join(strings.Fields(value), " ")
}
