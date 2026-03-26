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

func authStringPointer(value string) *string {
	return &value
}

func authInt64Pointer(value int64) *int64 {
	return &value
}

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
