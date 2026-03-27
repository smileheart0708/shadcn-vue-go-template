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

//go:fix inline
func authStringPointer(value string) *string {
	return new(value)
}

//go:fix inline
func authInt64Pointer(value int64) *int64 {
	return new(value)
}
