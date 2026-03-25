package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"main/internal/logging"
)

const systemLogsHeartbeatInterval = 25 * time.Second

func (api *API) adminStreamSystemLogsHandler(w http.ResponseWriter, r *http.Request) {
	logger := api.logger
	if logger == nil {
		logger = slog.Default()
	}

	if api.logStream == nil {
		logger.ErrorContext(
			r.Context(),
			"system log stream is unavailable",
			"operation", "stream_system_logs",
			"path", r.URL.Path,
		)
		writeAPIError(w, http.StatusServiceUnavailable, "system_log_stream_unavailable", "System log stream is unavailable.")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.ErrorContext(
			r.Context(),
			"response writer does not support flush",
			"operation", "stream_system_logs",
			"path", r.URL.Path,
		)
		writeAPIError(w, http.StatusInternalServerError, "streaming_unsupported", "Streaming is not supported by this server.")
		return
	}

	currentUser, ok := CurrentUser(r.Context())
	if !ok {
		logger.ErrorContext(
			r.Context(),
			"system log stream missing authenticated user",
			"operation", "stream_system_logs",
			"path", r.URL.Path,
		)
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	tail := parseSystemLogTail(r, logger)
	subscription := api.logStream.Subscribe()
	defer subscription.Close("client_disconnect")

	logger.InfoContext(
		r.Context(),
		"system log stream subscribed",
		"operation", "stream_system_logs",
		"path", r.URL.Path,
		"user_id", currentUser.UserID,
		"tail", tail,
		"subscriber_count", api.logStream.SubscriberCount(),
	)

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for _, entry := range api.logStream.Snapshot(tail) {
		if err := writeSystemLogSSEEvent(w, entry); err != nil {
			logger.WarnContext(
				r.Context(),
				"failed to replay system log entry",
				"operation", "stream_system_logs",
				"path", r.URL.Path,
				"user_id", currentUser.UserID,
				"entry_id", entry.ID,
				"error", err,
			)
			return
		}
		flusher.Flush()
	}

	heartbeat := time.NewTicker(systemLogsHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			logger.InfoContext(
				r.Context(),
				"system log stream closed",
				"operation", "stream_system_logs",
				"path", r.URL.Path,
				"user_id", currentUser.UserID,
				"reason", "client_disconnect",
			)
			return
		case <-heartbeat.C:
			if _, err := w.Write([]byte(": keep-alive\n\n")); err != nil {
				logger.WarnContext(
					r.Context(),
					"failed to write system log heartbeat",
					"operation", "stream_system_logs",
					"path", r.URL.Path,
					"user_id", currentUser.UserID,
					"error", err,
				)
				return
			}
			flusher.Flush()
		case entry, ok := <-subscription.Entries():
			if !ok {
				logger.WarnContext(
					r.Context(),
					"system log subscriber closed by broadcaster",
					"operation", "stream_system_logs",
					"path", r.URL.Path,
					"user_id", currentUser.UserID,
					"reason", subscription.Reason(),
				)
				return
			}
			if err := writeSystemLogSSEEvent(w, entry); err != nil {
				logger.WarnContext(
					r.Context(),
					"failed to write streamed system log entry",
					"operation", "stream_system_logs",
					"path", r.URL.Path,
					"user_id", currentUser.UserID,
					"entry_id", entry.ID,
					"error", err,
				)
				return
			}
			flusher.Flush()
		}
	}
}

func parseSystemLogTail(r *http.Request, logger *slog.Logger) int {
	rawTail := r.URL.Query().Get("tail")
	if rawTail == "" {
		return logging.DefaultReplayLimit
	}

	tail, err := strconv.Atoi(rawTail)
	if err != nil {
		logger.WarnContext(
			r.Context(),
			"invalid system log tail parameter",
			"operation", "stream_system_logs",
			"path", r.URL.Path,
			"tail", rawTail,
			"error", err,
		)
		return logging.DefaultReplayLimit
	}

	if tail <= 0 {
		logger.WarnContext(
			r.Context(),
			"system log tail below minimum; using default replay limit",
			"operation", "stream_system_logs",
			"path", r.URL.Path,
			"tail", tail,
		)
		return logging.DefaultReplayLimit
	}

	if tail > logging.DefaultStreamCapacity {
		logger.WarnContext(
			r.Context(),
			"system log tail exceeded maximum; clamping to stream capacity",
			"operation", "stream_system_logs",
			"path", r.URL.Path,
			"tail", tail,
			"max_tail", logging.DefaultStreamCapacity,
		)
		return logging.DefaultStreamCapacity
	}

	return tail
}

func writeSystemLogSSEEvent(w http.ResponseWriter, entry logging.StreamEntry) error {
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if _, err := w.Write([]byte("event: log\n")); err != nil {
		return err
	}
	if _, err := w.Write([]byte("id: " + strconv.FormatInt(entry.ID, 10) + "\n")); err != nil {
		return err
	}
	if _, err := w.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	_, err = w.Write([]byte("\n\n"))
	return err
}
