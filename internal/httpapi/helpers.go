package httpapi

import (
	"net/http"
	"strings"
)

func requestMetadata(r *http.Request) (string, string) {
	return clientAddr(r.RemoteAddr), strings.TrimSpace(r.UserAgent())
}
