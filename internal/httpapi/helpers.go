package httpapi

import (
	"net/http"
	"strings"
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
