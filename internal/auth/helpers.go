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

	sum := sha256.Sum256(options.Secret)
	return "refresh_token_" + hex.EncodeToString(sum[:6])
}

func BearerTokenFromHeader(header string) (string, error) {
	if header == "" {
		return "", ErrMissingBearerToken
	}

	prefix, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(prefix, "Bearer") {
		return "", ErrMissingBearerToken
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrMissingBearerToken
	}
	return token, nil
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

func (s *Service) SetRefreshCookie(w http.ResponseWriter, _ *http.Request, refreshToken string) {
	http.SetCookie(w, s.newRefreshCookie(refreshToken, int(s.refreshAbsoluteTTL.Seconds())))
}

func (s *Service) ClearRefreshCookie(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, s.newRefreshCookie("", -1))
}

func (s *Service) newRefreshCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     s.refreshCookieName,
		Value:    value,
		Path:     RefreshCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	}
}

func randomTokenComponent(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("auth: generate secure token: %w", err)
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
