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

func (api *API) streamSystemLogsHandler(w http.ResponseWriter, r *http.Request) {
	logger := api.logger
	if logger == nil {
		logger = slog.Default()
	}

	if api.logStream == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "system_log_stream_unavailable", "System log stream is unavailable.")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "streaming_unsupported", "Streaming is not supported by this server.")
		return
	}

	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	tail := parseSystemLogTail(r, logger)
	subscription := api.logStream.Subscribe()
	defer subscription.Close("client_disconnect")

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for _, entry := range api.logStream.Snapshot(tail) {
		if err := writeSystemLogSSEEvent(w, entry); err != nil {
			logger.WarnContext(r.Context(), "failed to replay system log entry", "user_id", actor.User.ID, "entry_id", entry.ID, "error", err)
			return
		}
		flusher.Flush()
	}

	heartbeat := time.NewTicker(systemLogsHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := w.Write([]byte(": keep-alive\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case entry, ok := <-subscription.Entries():
			if !ok {
				return
			}
			if err := writeSystemLogSSEEvent(w, entry); err != nil {
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
		logger.WarnContext(r.Context(), "invalid system log tail parameter", "tail", rawTail, "error", err)
		return logging.DefaultReplayLimit
	}
	if tail <= 0 {
		return logging.DefaultReplayLimit
	}
	if tail > logging.DefaultStreamCapacity {
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
