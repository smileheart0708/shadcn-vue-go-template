package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

const RefreshCookiePath = "/api/auth"

func ResolveRefreshCookieName(options Options) string {
	if cookieName := strings.TrimSpace(options.RefreshCookieName); cookieName != "" {
		return cookieName
	}

	// Browsers scope cookies by host and path, not port. Namespacing the default
	// refresh cookie by the JWT secret keeps localhost apps from overwriting each
	// other's sessions when they all run on localhost:8080.
	sum := sha256.Sum256(options.Secret)
	return "refresh_token_" + hex.EncodeToString(sum[:6])
}

func (s *Service) ReadRefreshCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(s.refreshCookieName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cookie.Value) == "" {
		return "", http.ErrNoCookie
	}

	return cookie.Value, nil
}

func (s *Service) SetRefreshCookie(w http.ResponseWriter, r *http.Request, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.refreshCookieName,
		Value:    refreshToken,
		Path:     RefreshCookiePath,
		MaxAge:   int(s.refreshAbsoluteTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(r),
	})
}

func (s *Service) ClearRefreshCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.refreshCookieName,
		Value:    "",
		Path:     RefreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(r),
	})
}

func randomTokenComponent(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	return hex.EncodeToString(buffer), nil
}

func composeRefreshToken(sessionID string, secret string) string {
	return sessionID + "." + secret
}

func ParseRefreshToken(rawToken string) (string, string, error) {
	sessionID, secret, ok := strings.Cut(strings.TrimSpace(rawToken), ".")
	if !ok || sessionID == "" || secret == "" {
		return "", "", ErrInvalidRefreshToken
	}

	return sessionID, secret, nil
}

func hashRefreshToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func requestUsesHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}

	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func userAuthVersionCacheKey(userID int64) string {
	return fmt.Sprintf("user_auth_version:%d", userID)
}

func refreshSessionCacheKey(sessionID string) string {
	return "refresh_session:" + sessionID
}

//go:fix inline
func stringPtr(value string) *string {
	return new(value)
}

//go:fix inline
func int64Ptr(value int64) *int64 {
	return new(value)
}

func stringPtrOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return new(value)
}
