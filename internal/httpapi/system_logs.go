package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"main/internal/logging"
)

const systemLogsHeartbeatInterval = 25 * time.Second

func (api *API) streamSystemLogsHandler(w http.ResponseWriter, r *http.Request) {
	logger := api.loggerOrDefault()

	if api.logStream == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "system_log_stream_unavailable", "System log stream is unavailable.")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "streaming_unsupported", "Streaming is not supported by this server.")
		return
	}

	actor, authenticated := api.currentActor(r.Context())
	if !authenticated {
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
				logger.WarnContext(r.Context(), "failed to write system log heartbeat", "user_id", actor.User.ID, "error", err)
				return
			}
			flusher.Flush()
		case streamEntry, entriesOK := <-subscription.Entries():
			if !entriesOK {
				return
			}
			if err := writeSystemLogSSEEvent(w, streamEntry); err != nil {
				logger.WarnContext(r.Context(), "failed to stream system log entry", "user_id", actor.User.ID, "entry_id", streamEntry.ID, "error", err)
				return
			}
			flusher.Flush()
		}
	}
}

func parseSystemLogTail(r *http.Request, logger *slog.Logger) int {
	rawTail := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("tail")))
	if rawTail == "" || rawTail == "all" {
		return 0
	}

	tail, err := strconv.Atoi(rawTail)
	if err != nil {
		logger.WarnContext(r.Context(), "invalid system log tail parameter", "tail", rawTail, "error", err)
		return 0
	}
	switch tail {
	case 100, 200, 500, 1000:
		return tail
	default:
		logger.WarnContext(r.Context(), "unsupported system log tail parameter", "tail", rawTail)
		return 0
	}
}

func writeSystemLogSSEEvent(w http.ResponseWriter, entry logging.StreamEntry) error {
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if _, writeErr := w.Write([]byte("event: log\n")); writeErr != nil {
		return writeErr
	}
	if _, writeErr := w.Write([]byte("id: " + strconv.FormatInt(entry.ID, 10) + "\n")); writeErr != nil {
		return writeErr
	}
	if _, writeErr := w.Write([]byte("data: ")); writeErr != nil {
		return writeErr
	}
	if _, writeErr := w.Write(payload); writeErr != nil {
		return writeErr
	}
	_, err = w.Write([]byte("\n\n"))
	return err
}
