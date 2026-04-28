package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"main/internal/audit"
)

func requestMetadata(r *http.Request) (string, string) {
	return clientAddr(r.RemoteAddr), strings.TrimSpace(r.UserAgent())
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (api *API) logAuditEvent(r *http.Request, entry audit.Entry) {
	if api == nil || api.audit == nil {
		return
	}

	if entry.IP == nil || entry.UserAgent == nil {
		ip, userAgent := requestMetadata(r)
		if entry.IP == nil {
			entry.IP = nullableString(ip)
		}
		if entry.UserAgent == nil {
			entry.UserAgent = nullableString(userAgent)
		}
	}

	if err := api.audit.Log(r.Context(), entry); err != nil {
		slog.ErrorContext(r.Context(), "failed to write audit event", "event_type", entry.EventType, "outcome", entry.Outcome, "error", err)
	}
}
