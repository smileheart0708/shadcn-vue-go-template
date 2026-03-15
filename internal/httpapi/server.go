package httpapi

import (
	"bytes"
	"database/sql"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

type HandlerOptions struct {
	Logger         *slog.Logger
	DB             *sql.DB
	DataDir        string
	FrontendFS     fs.FS
	Auth           AuthOptions
	LogAPIRequests bool
}

func NewHandlerWithOptions(options HandlerOptions) http.Handler {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	apiService := NewAPI(options.DB, options.DataDir, options.Auth)
	api := newAPIMux(apiService)
	if options.LogAPIRequests {
		api = Chain(api, requestLogger(logger))
	}
	spa := newSPAHandler(options.FrontendFS)

	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIRequest(r.URL.Path) {
			api.ServeHTTP(w, r)
			return
		}

		spa.ServeHTTP(w, r)
	})

	return root
}

func newAPIMux(api *API) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", healthz)
	mux.Handle("POST /api/auth/login", http.HandlerFunc(api.loginHandler))
	mux.Handle("GET /api/auth/me", Chain(http.HandlerFunc(api.meHandler), RequireAuth(api.auth)))
	mux.Handle("PATCH /api/account/profile", Chain(http.HandlerFunc(api.updateProfileHandler), RequireAuth(api.auth)))
	mux.Handle("POST /api/account/avatar", Chain(http.HandlerFunc(api.updateAvatarHandler), RequireAuth(api.auth)))
	mux.Handle("POST /api/account/password", Chain(http.HandlerFunc(api.updatePasswordHandler), RequireAuth(api.auth)))
	mux.Handle("DELETE /api/account", Chain(http.HandlerFunc(api.deleteAccountHandler), RequireAuth(api.auth)))
	mux.Handle("GET /api/avatars/{filename}", http.HandlerFunc(api.avatarHandler))
	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

type spaHandler struct {
	fsys fs.FS
}

func newSPAHandler(fsys fs.FS) http.Handler {
	if fsys == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "frontend assets are unavailable", http.StatusServiceUnavailable)
		})
	}

	return &spaHandler{fsys: fsys}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}

	requestedPath := cleanFrontendPath(r.URL.Path)
	switch {
	case requestedPath == "" || requestedPath == ".":
		h.serveFrontendFile(w, r, "index.html")
	case h.fileExists(requestedPath):
		h.serveFrontendFile(w, r, requestedPath)
	case looksLikeStaticAsset(requestedPath):
		http.NotFound(w, r)
	default:
		h.serveFrontendFile(w, r, "index.html")
	}
}

func (h *spaHandler) serveFrontendFile(w http.ResponseWriter, r *http.Request, name string) {
	if clientAcceptsGzip(r.Header.Get("Accept-Encoding")) && h.fileExists(name+".gz") {
		setVary(w.Header(), "Accept-Encoding")
		w.Header().Set("Content-Encoding", "gzip")
		serveFSContent(w, r, h.fsys, name+".gz", name)
		return
	}

	serveFSContent(w, r, h.fsys, name, name)
}

func (h *spaHandler) fileExists(name string) bool {
	info, err := fs.Stat(h.fsys, name)
	return err == nil && !info.IsDir()
}

func serveFSContent(w http.ResponseWriter, r *http.Request, fsys fs.FS, storedName string, originalName string) {
	data, err := fs.ReadFile(fsys, storedName)
	if err != nil {
		status := http.StatusNotFound
		if originalName == "index.html" {
			status = http.StatusServiceUnavailable
		}

		http.Error(w, http.StatusText(status), status)
		return
	}

	if contentType := mime.TypeByExtension(path.Ext(originalName)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	http.ServeContent(w, r, originalName, time.Time{}, bytes.NewReader(data))
}

func isAPIRequest(requestPath string) bool {
	return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/")
}

func cleanFrontendPath(requestPath string) string {
	cleaned := path.Clean("/" + requestPath)
	return strings.TrimPrefix(cleaned, "/")
}

func looksLikeStaticAsset(name string) bool {
	return strings.Contains(path.Base(name), ".")
}

func clientAcceptsGzip(header string) bool {
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		encoding, params, _ := strings.Cut(part, ";")
		encoding = strings.ToLower(strings.TrimSpace(encoding))
		if encoding != "gzip" && encoding != "*" {
			continue
		}

		if qualityValueAllowed(params) {
			return true
		}
	}

	return false
}

func qualityValueAllowed(params string) bool {
	params = strings.TrimSpace(params)
	if params == "" {
		return true
	}

	for _, param := range strings.Split(params, ";") {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}

		key, value, ok := strings.Cut(param, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") {
			continue
		}

		quality, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return false
		}

		return quality > 0
	}

	return true
}

func setVary(header http.Header, value string) {
	if current := header.Values("Vary"); len(current) > 0 {
		for _, entry := range current {
			for _, item := range strings.Split(entry, ",") {
				if strings.EqualFold(strings.TrimSpace(item), value) {
					return
				}
			}
		}
	}

	header.Add("Vary", value)
}

func requestLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(recorder, r)

			route := r.Pattern
			if route == "" {
				route = r.URL.Path
			}

			logger.InfoContext(
				r.Context(),
				"http request",
				"method", r.Method,
				"route", route,
				"status", recorder.status,
				"duration", time.Since(start),
				"remote_addr", clientAddr(r.RemoteAddr),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	return r.ResponseWriter.Write(p)
}

func clientAddr(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}

	return remoteAddr
}
