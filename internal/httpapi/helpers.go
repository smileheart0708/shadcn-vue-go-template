package httpapi

import (
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

	_ = api.audit.Log(r.Context(), entry)
}
