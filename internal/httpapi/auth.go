package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickmn/go-cache"

	"main/internal/logging"
	"main/internal/users"
)

const (
	minPasswordLength       = 8
	maxAvatarFileSizeBytes  = 5 * 1024 * 1024
	maxAvatarUploadBodySize = maxAvatarFileSizeBytes + (1 << 20)
	refreshCookieName       = "refresh_token"
	refreshCookiePath       = "/api/auth"
	defaultCacheTTL         = 5 * time.Minute
	sessionCacheTTL         = 2 * time.Minute
)

var ErrMissingBearerToken = errors.New("missing bearer token")
var ErrInvalidToken = errors.New("invalid token")
var ErrInvalidRefreshToken = errors.New("invalid refresh token")
var ErrRefreshTokenExpired = errors.New("refresh token expired")
var ErrRefreshTokenRevoked = errors.New("refresh token revoked")
var ErrTokenReuseDetected = errors.New("refresh token reuse detected")

type AuthOptions struct {
	Issuer             string
	Secret             []byte
	TTL                time.Duration
	RefreshIdleTTL     time.Duration
	RefreshAbsoluteTTL time.Duration
}

type API struct {
	users     *users.Store
	auth      *AuthService
	dataDir   string
	logger    *slog.Logger
	logStream *logging.Stream
}

type AuthService struct {
	issuer             string
	secret             []byte
	ttl                time.Duration
	refreshIdleTTL     time.Duration
	refreshAbsoluteTTL time.Duration
	users              *users.Store
	authVersionCache   *cache.Cache
	sessionCache       *cache.Cache
}

type AuthPrincipal struct {
	UserID      int64
	SessionID   string
	Role        int
	AuthVersion int64
}

type authClaims struct {
	SessionID   string `json:"sid"`
	Role        int    `json:"role"`
	AuthVersion int64  `json:"ver"`
	jwt.RegisteredClaims
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type loginResponse struct {
	AccessToken string           `json:"accessToken"`
	TokenType   string           `json:"tokenType"`
	ExpiresAt   time.Time        `json:"expiresAt"`
	User        users.PublicUser `json:"user"`
}

type logoutResponse struct {
	LoggedOut bool `json:"loggedOut"`
}

type updateProfileRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email"`
}

type updatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type deleteAccountResponse struct {
	Deleted bool `json:"deleted"`
}

type authContextKey string

type refreshResult struct {
	User         users.User
	SessionID    string
	RefreshToken string
	AccessToken  string
	ExpiresAt    time.Time
}

type refreshAttempt struct {
	UserID        *int64
	SessionID     *string
	FailureReason string
}

type cachedSessionState struct {
	UserID        int64
	TokenHash     string
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
	RevokedAt     *time.Time
}

const authUserContextKey authContextKey = "auth-user"

func NewAPI(db *sql.DB, dataDir string, authOptions AuthOptions) *API {
	userStore := users.NewStore(db)
	return &API{
		users:   userStore,
		auth:    NewAuthService(authOptions, userStore),
		dataDir: dataDir,
	}
}

func NewAuthService(options AuthOptions, userStore *users.Store) *AuthService {
	if options.Issuer == "" {
		options.Issuer = "shadcn-vue-go-template"
	}
	if len(options.Secret) == 0 {
		options.Secret = []byte("change-me-in-production")
	}
	if options.TTL <= 0 {
		options.TTL = 10 * time.Minute
	}
	if options.RefreshIdleTTL <= 0 {
		options.RefreshIdleTTL = 7 * 24 * time.Hour
	}
	if options.RefreshAbsoluteTTL <= 0 {
		options.RefreshAbsoluteTTL = 30 * 24 * time.Hour
	}

	return &AuthService{
		issuer:             options.Issuer,
		secret:             options.Secret,
		ttl:                options.TTL,
		refreshIdleTTL:     options.RefreshIdleTTL,
		refreshAbsoluteTTL: options.RefreshAbsoluteTTL,
		users:              userStore,
		authVersionCache:   cache.New(defaultCacheTTL, defaultCacheTTL),
		sessionCache:       cache.New(sessionCacheTTL, sessionCacheTTL),
	}
}

func (s *AuthService) IssueToken(user users.User, sessionID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)

	claims := authClaims{
		SessionID:   sessionID,
		Role:        user.Role,
		AuthVersion: user.AuthVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	encoded, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return encoded, expiresAt, nil
}

func (s *AuthService) ParseToken(token string) (AuthPrincipal, error) {
	parsed, err := jwt.ParseWithClaims(token, &authClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidToken
		}

		return s.secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return AuthPrincipal{}, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*authClaims)
	if !ok {
		return AuthPrincipal{}, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 || claims.AuthVersion <= 0 || strings.TrimSpace(claims.SessionID) == "" {
		return AuthPrincipal{}, ErrInvalidToken
	}

	return AuthPrincipal{
		UserID:      userID,
		SessionID:   claims.SessionID,
		Role:        claims.Role,
		AuthVersion: claims.AuthVersion,
	}, nil
}

func (s *AuthService) ValidatePrincipal(ctx context.Context, principal AuthPrincipal) error {
	authVersion, err := s.loadAuthVersion(ctx, principal.UserID)
	if err != nil {
		return err
	}
	if authVersion != principal.AuthVersion {
		return ErrInvalidToken
	}

	sessionState, err := s.loadSessionState(ctx, principal.SessionID)
	if err != nil {
		return err
	}
	if sessionState.UserID != principal.UserID || sessionState.RevokedAt != nil {
		return ErrInvalidToken
	}

	now := time.Now().UTC()
	if now.After(sessionState.ExpiresAt) || now.After(sessionState.IdleExpiresAt) {
		return ErrInvalidToken
	}

	return nil
}

func (s *AuthService) CreateRefreshSession(ctx context.Context, user users.User, ip string, userAgent string) (string, string, error) {
	now := time.Now().UTC()
	sessionID, err := randomTokenComponent(16)
	if err != nil {
		return "", "", err
	}
	sessionSecret, err := randomTokenComponent(32)
	if err != nil {
		return "", "", err
	}

	refreshToken := composeRefreshToken(sessionID, sessionSecret)
	tokenHash := hashRefreshToken(refreshToken)
	session := users.CreateRefreshSessionParams{
		ID:            sessionID,
		UserID:        user.ID,
		TokenHash:     tokenHash,
		IssuedAt:      now,
		LastUsedAt:    now,
		ExpiresAt:     now.Add(s.refreshAbsoluteTTL),
		IdleExpiresAt: now.Add(s.refreshIdleTTL),
		IP:            nullableString(ip),
		UserAgent:     nullableString(userAgent),
	}

	if err := s.users.CreateRefreshSession(ctx, session); err != nil {
		return "", "", err
	}

	s.setSessionState(session.ID, cachedSessionState{
		UserID:        session.UserID,
		TokenHash:     session.TokenHash,
		ExpiresAt:     session.ExpiresAt,
		IdleExpiresAt: session.IdleExpiresAt,
	})

	return session.ID, refreshToken, nil
}

func (s *AuthService) RefreshSession(ctx context.Context, rawToken string) (refreshResult, refreshAttempt, error) {
	sessionID, _, err := parseRefreshToken(rawToken)
	if err != nil {
		return refreshResult{}, refreshAttempt{FailureReason: "refresh_token_invalid"}, ErrInvalidRefreshToken
	}

	attempt := refreshAttempt{
		SessionID:     authStringPointer(sessionID),
		FailureReason: "refresh_failed",
	}

	sessionState, err := s.loadSessionState(ctx, sessionID)
	if err != nil {
		if errors.Is(err, users.ErrRefreshSessionNotFound) {
			attempt.FailureReason = "refresh_session_not_found"
			return refreshResult{}, attempt, ErrInvalidRefreshToken
		}
		return refreshResult{}, attempt, err
	}

	attempt.UserID = authInt64Pointer(sessionState.UserID)

	tokenHash := hashRefreshToken(rawToken)
	if subtle.ConstantTimeCompare([]byte(sessionState.TokenHash), []byte(tokenHash)) != 1 {
		attempt.FailureReason = "token_reuse_detected"
		_ = s.revokeRefreshSession(ctx, sessionID, "token_reuse_detected")
		return refreshResult{}, attempt, ErrTokenReuseDetected
	}

	now := time.Now().UTC()
	switch {
	case sessionState.RevokedAt != nil:
		attempt.FailureReason = "refresh_session_revoked"
		return refreshResult{}, attempt, ErrRefreshTokenRevoked
	case now.After(sessionState.ExpiresAt):
		attempt.FailureReason = "refresh_token_expired"
		_ = s.revokeRefreshSession(ctx, sessionID, "refresh_token_expired")
		return refreshResult{}, attempt, ErrRefreshTokenExpired
	case now.After(sessionState.IdleExpiresAt):
		attempt.FailureReason = "refresh_token_idle_expired"
		_ = s.revokeRefreshSession(ctx, sessionID, "refresh_token_idle_expired")
		return refreshResult{}, attempt, ErrRefreshTokenExpired
	}

	user, err := s.users.GetByID(ctx, sessionState.UserID)
	if err != nil {
		attempt.FailureReason = "user_not_found"
		_ = s.revokeRefreshSession(ctx, sessionID, "user_not_found")
		return refreshResult{}, attempt, ErrInvalidRefreshToken
	}

	authVersion, err := s.loadAuthVersion(ctx, user.ID)
	if err != nil {
		return refreshResult{}, attempt, err
	}
	user.AuthVersion = authVersion

	nextSecret, err := randomTokenComponent(32)
	if err != nil {
		return refreshResult{}, attempt, err
	}
	nextRefreshToken := composeRefreshToken(sessionID, nextSecret)
	nextTokenHash := hashRefreshToken(nextRefreshToken)
	nextIdleExpiry := now.Add(s.refreshIdleTTL)

	if err := s.users.RotateRefreshSession(ctx, users.RotateRefreshSessionParams{
		ID:            sessionID,
		TokenHash:     nextTokenHash,
		LastUsedAt:    now,
		IdleExpiresAt: nextIdleExpiry,
	}); err != nil {
		return refreshResult{}, attempt, err
	}

	s.setSessionState(sessionID, cachedSessionState{
		UserID:        user.ID,
		TokenHash:     nextTokenHash,
		ExpiresAt:     sessionState.ExpiresAt,
		IdleExpiresAt: nextIdleExpiry,
	})

	accessToken, expiresAt, err := s.IssueToken(user, sessionID)
	if err != nil {
		return refreshResult{}, attempt, err
	}

	return refreshResult{
		User:         user,
		SessionID:    sessionID,
		RefreshToken: nextRefreshToken,
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
	}, attempt, nil
}

func (s *AuthService) RevokeRefreshSession(ctx context.Context, rawToken string, reason string) (*string, error) {
	sessionID, _, err := parseRefreshToken(rawToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if err := s.revokeRefreshSession(ctx, sessionID, reason); err != nil {
		return authStringPointer(sessionID), err
	}

	return authStringPointer(sessionID), nil
}

func (s *AuthService) RevokeRefreshSessionsByUser(ctx context.Context, userID int64, reason string) error {
	if _, err := s.users.RevokeRefreshSessionsByUser(ctx, userID, reason, time.Now().UTC()); err != nil {
		return err
	}

	s.sessionCache.Flush()
	return nil
}

func (s *AuthService) InvalidateUser(userID int64) {
	s.authVersionCache.Delete(userAuthVersionCacheKey(userID))
}

func (s *AuthService) RevokeSessionByID(ctx context.Context, sessionID string, reason string) error {
	return s.revokeRefreshSession(ctx, sessionID, reason)
}

func (s *AuthService) loadAuthVersion(ctx context.Context, userID int64) (int64, error) {
	cacheKey := userAuthVersionCacheKey(userID)
	if cached, ok := s.authVersionCache.Get(cacheKey); ok {
		if authVersion, ok := cached.(int64); ok {
			return authVersion, nil
		}
	}

	authVersion, err := s.users.GetAuthVersion(ctx, userID)
	if err != nil {
		return 0, err
	}

	s.authVersionCache.Set(cacheKey, authVersion, cache.DefaultExpiration)
	return authVersion, nil
}

func (s *AuthService) loadSessionState(ctx context.Context, sessionID string) (cachedSessionState, error) {
	cacheKey := refreshSessionCacheKey(sessionID)
	if cached, ok := s.sessionCache.Get(cacheKey); ok {
		if state, ok := cached.(cachedSessionState); ok {
			return state, nil
		}
	}

	session, err := s.users.GetRefreshSessionByID(ctx, sessionID)
	if err != nil {
		return cachedSessionState{}, err
	}

	state := cachedSessionState{
		UserID:        session.UserID,
		TokenHash:     session.TokenHash,
		ExpiresAt:     session.ExpiresAt,
		IdleExpiresAt: session.IdleExpiresAt,
		RevokedAt:     session.RevokedAt,
	}
	s.setSessionState(sessionID, state)
	return state, nil
}

func (s *AuthService) setSessionState(sessionID string, state cachedSessionState) {
	s.sessionCache.Set(refreshSessionCacheKey(sessionID), state, cache.DefaultExpiration)
}

func (s *AuthService) revokeRefreshSession(ctx context.Context, sessionID string, reason string) error {
	err := s.users.RevokeRefreshSession(ctx, sessionID, reason, time.Now().UTC())
	if err != nil && !errors.Is(err, users.ErrRefreshSessionNotFound) {
		return err
	}

	s.sessionCache.Delete(refreshSessionCacheKey(sessionID))
	return nil
}

func RequireAuth(auth *AuthService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerTokenFromHeader(r.Header.Get("Authorization"))
			if err != nil {
				writeAPIError(w, http.StatusUnauthorized, "missing_token", "Missing bearer token.")
				return
			}

			principal, err := auth.ParseToken(token)
			if err != nil {
				writeAPIError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired token.")
				return
			}

			if err := auth.ValidatePrincipal(r.Context(), principal); err != nil {
				writeAPIError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired token.")
				return
			}

			ctx := context.WithValue(r.Context(), authUserContextKey, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(minRole int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := CurrentUser(r.Context())
			if !ok {
				writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
				return
			}

			if principal.Role < minRole {
				writeAPIError(w, http.StatusForbidden, "forbidden", "You do not have permission to access this resource.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CurrentUser(ctx context.Context) (AuthPrincipal, bool) {
	principal, ok := ctx.Value(authUserContextKey).(AuthPrincipal)
	return principal, ok
}

func (api *API) loginHandler(w http.ResponseWriter, r *http.Request) {
	var payload loginRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}

	identifier := strings.TrimSpace(payload.Identifier)
	requestIP, requestUserAgent := authRequestMetadata(r)

	user, err := api.users.FindByIdentifier(r.Context(), identifier)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			api.logAuthEvent(r.Context(), users.AuthLogParams{
				Identifier:    nullableString(identifier),
				IP:            nullableString(requestIP),
				UserAgent:     nullableString(requestUserAgent),
				EventType:     "login_failed",
				Success:       false,
				FailureReason: authStringPointer("invalid_credentials"),
			})
			writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Username/email or password is invalid.")
			return
		}

		writeAPIError(w, http.StatusInternalServerError, "login_failed", "Failed to authenticate user.")
		return
	}

	passwordMatches, err := users.VerifyPassword(payload.Password, user.PasswordHash)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "login_failed", "Failed to authenticate user.")
		return
	}
	if !passwordMatches {
		api.logAuthEvent(r.Context(), users.AuthLogParams{
			UserID:        authInt64Pointer(user.ID),
			Identifier:    nullableString(identifier),
			IP:            nullableString(requestIP),
			UserAgent:     nullableString(requestUserAgent),
			EventType:     "login_failed",
			Success:       false,
			FailureReason: authStringPointer("invalid_credentials"),
		})
		writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Username/email or password is invalid.")
		return
	}

	sessionID, refreshToken, err := api.auth.CreateRefreshSession(r.Context(), user, requestIP, requestUserAgent)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_create_failed", "Failed to create refresh session.")
		return
	}

	accessToken, expiresAt, err := api.auth.IssueToken(user, sessionID)
	if err != nil {
		_ = api.auth.RevokeSessionByID(r.Context(), sessionID, "access_token_issue_failed")
		writeAPIError(w, http.StatusInternalServerError, "token_issue_failed", "Failed to issue access token.")
		return
	}

	setRefreshCookie(w, r, refreshToken, api.auth.refreshAbsoluteTTL)
	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:     authInt64Pointer(user.ID),
		SessionID:  authStringPointer(sessionID),
		Identifier: nullableString(identifier),
		IP:         nullableString(requestIP),
		UserAgent:  nullableString(requestUserAgent),
		EventType:  "login_success",
		Success:    true,
	})

	writeSuccessJSON(w, http.StatusOK, loginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        user.Public(),
	})
}

func (api *API) refreshHandler(w http.ResponseWriter, r *http.Request) {
	rawToken, err := readRefreshCookie(r)
	if err != nil {
		api.logAuthEvent(r.Context(), users.AuthLogParams{
			IP:            nullableString(clientAddr(r.RemoteAddr)),
			UserAgent:     nullableString(r.UserAgent()),
			EventType:     "refresh_failed",
			Success:       false,
			FailureReason: authStringPointer("refresh_cookie_missing"),
		})
		clearRefreshCookie(w, r)
		writeAPIError(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh session is invalid or expired.")
		return
	}

	result, attempt, err := api.auth.RefreshSession(r.Context(), rawToken)
	requestIP, requestUserAgent := authRequestMetadata(r)
	if err != nil {
		eventType := "refresh_failed"
		if errors.Is(err, ErrTokenReuseDetected) {
			eventType = "token_reuse_detected"
		}
		api.logAuthEvent(r.Context(), users.AuthLogParams{
			UserID:        attempt.UserID,
			SessionID:     attempt.SessionID,
			IP:            nullableString(requestIP),
			UserAgent:     nullableString(requestUserAgent),
			EventType:     eventType,
			Success:       false,
			FailureReason: authStringPointer(attempt.FailureReason),
		})
		clearRefreshCookie(w, r)
		writeAPIError(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh session is invalid or expired.")
		return
	}

	setRefreshCookie(w, r, result.RefreshToken, api.auth.refreshAbsoluteTTL)
	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:    authInt64Pointer(result.User.ID),
		SessionID: authStringPointer(result.SessionID),
		IP:        nullableString(requestIP),
		UserAgent: nullableString(requestUserAgent),
		EventType: "refresh_success",
		Success:   true,
	})

	writeSuccessJSON(w, http.StatusOK, loginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   result.ExpiresAt,
		User:        result.User.Public(),
	})
}

func (api *API) logoutHandler(w http.ResponseWriter, r *http.Request) {
	requestIP, requestUserAgent := authRequestMetadata(r)
	rawToken, err := readRefreshCookie(r)
	if err == nil {
		sessionID, _, parseErr := parseRefreshToken(rawToken)
		if parseErr == nil {
			sessionState, stateErr := api.auth.loadSessionState(r.Context(), sessionID)
			if stateErr == nil {
				api.logAuthEvent(r.Context(), users.AuthLogParams{
					UserID:    authInt64Pointer(sessionState.UserID),
					SessionID: authStringPointer(sessionID),
					IP:        nullableString(requestIP),
					UserAgent: nullableString(requestUserAgent),
					EventType: "logout",
					Success:   true,
				})
			}
		}

		_, _ = api.auth.RevokeRefreshSession(r.Context(), rawToken, "logout")
	}

	clearRefreshCookie(w, r)
	writeSuccessJSON(w, http.StatusOK, logoutResponse{LoggedOut: true})
}

func (api *API) meHandler(w http.ResponseWriter, r *http.Request) {
	user, err := api.currentUser(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	writeSuccessJSON(w, http.StatusOK, user.Public())
}

func (api *API) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := api.currentUser(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	var payload updateProfileRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}
	if strings.TrimSpace(payload.Username) == "" {
		writeAPIError(w, http.StatusBadRequest, "username_required", "Username is required.")
		return
	}

	user, err := api.users.UpdateProfile(r.Context(), currentUser.ID, payload.Username, payload.Email)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrUsernameTaken):
			writeAPIError(w, http.StatusConflict, "username_taken", "Username is already in use.")
		case errors.Is(err, users.ErrEmailTaken):
			writeAPIError(w, http.StatusConflict, "email_taken", "Email is already in use.")
		case errors.Is(err, users.ErrUserNotFound):
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "profile_update_failed", "Failed to update account profile.")
		}
		return
	}

	writeSuccessJSON(w, http.StatusOK, user.Public())
}

func (api *API) updateAvatarHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := api.currentUser(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	currentAvatarPath := ""
	if currentUser.AvatarPath != nil {
		currentAvatarPath = filepath.Join(users.AvatarDir(api.dataDir), filepath.Base(*currentUser.AvatarPath))
	}

	avatarFileName, err := api.storeAvatarUpload(w, r)
	if err != nil {
		var avatarError *avatarUploadError
		if errors.As(err, &avatarError) {
			writeAPIError(w, avatarError.StatusCode, avatarError.Code, avatarError.Message)
			return
		}

		writeAPIError(w, http.StatusInternalServerError, "avatar_upload_failed", "Failed to upload avatar.")
		return
	}

	user, err := api.users.UpdateAvatar(r.Context(), currentUser.ID, &avatarFileName)
	if err != nil {
		_ = os.Remove(filepath.Join(users.AvatarDir(api.dataDir), avatarFileName))
		if errors.Is(err, users.ErrUserNotFound) {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "avatar_update_failed", "Failed to update avatar.")
		return
	}

	if currentAvatarPath != "" {
		_ = os.Remove(currentAvatarPath)
	}

	writeSuccessJSON(w, http.StatusOK, user.Public())
}

func (api *API) updatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := api.currentUser(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}
	principal, _ := CurrentUser(r.Context())

	var payload updatePasswordRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}
	if len(payload.NewPassword) < minPasswordLength {
		writeAPIError(w, http.StatusBadRequest, "password_too_short", "New password must be at least 8 characters.")
		return
	}

	passwordMatches, err := users.VerifyPassword(payload.CurrentPassword, currentUser.PasswordHash)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "password_update_failed", "Failed to update password.")
		return
	}
	if !passwordMatches {
		writeAPIError(w, http.StatusBadRequest, "current_password_invalid", "Current password is incorrect.")
		return
	}

	passwordHash, err := users.HashPassword(payload.NewPassword)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "password_update_failed", "Failed to update password.")
		return
	}

	user, err := api.users.UpdatePassword(r.Context(), currentUser.ID, passwordHash, false)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "password_update_failed", "Failed to update password.")
		return
	}

	if err := api.auth.RevokeRefreshSessionsByUser(r.Context(), user.ID, "password_changed"); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "password_update_failed", "Failed to revoke active sessions after password change.")
		return
	}
	api.auth.InvalidateUser(user.ID)

	requestIP, requestUserAgent := authRequestMetadata(r)
	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:    authInt64Pointer(user.ID),
		SessionID: nullableString(principal.SessionID),
		IP:        nullableString(requestIP),
		UserAgent: nullableString(requestUserAgent),
		EventType: "password_changed",
		Success:   true,
	})
	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:        authInt64Pointer(user.ID),
		SessionID:     nullableString(principal.SessionID),
		IP:            nullableString(requestIP),
		UserAgent:     nullableString(requestUserAgent),
		EventType:     "password_changed_forced_logout",
		Success:       true,
		FailureReason: authStringPointer("all_sessions_revoked"),
	})

	if err := users.ClearBootstrapPasswordFile(api.dataDir); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "bootstrap_password_cleanup_failed", "Password was changed, but cleanup failed.")
		return
	}

	clearRefreshCookie(w, r)
	writeSuccessJSON(w, http.StatusOK, user.Public())
}

func (api *API) deleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := api.currentUser(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}
	if currentUser.Role == users.RoleSuperAdmin {
		writeAPIError(w, http.StatusForbidden, "super_admin_delete_forbidden", "Super administrators cannot delete their own account.")
		return
	}

	if err := api.auth.RevokeRefreshSessionsByUser(r.Context(), currentUser.ID, "account_deleted"); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "account_delete_failed", "Failed to revoke active sessions.")
		return
	}
	api.auth.InvalidateUser(currentUser.ID)

	if err := api.users.Delete(r.Context(), currentUser.ID); err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "account_delete_failed", "Failed to delete account.")
		return
	}

	if currentUser.AvatarPath != nil {
		_ = os.Remove(filepath.Join(users.AvatarDir(api.dataDir), filepath.Base(*currentUser.AvatarPath)))
	}

	clearRefreshCookie(w, r)
	writeSuccessJSON(w, http.StatusOK, deleteAccountResponse{Deleted: true})
}

func (api *API) avatarHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if filename == "" || filename != filepath.Base(filename) || strings.Contains(filename, `\`) {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join(users.AvatarDir(api.dataDir), filename)
	if _, err := os.Stat(fullPath); err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, fullPath)
}

func (api *API) currentUser(ctx context.Context) (users.User, error) {
	principal, ok := CurrentUser(ctx)
	if !ok {
		return users.User{}, users.ErrUserNotFound
	}

	user, err := api.users.GetByID(ctx, principal.UserID)
	if err != nil {
		return users.User{}, err
	}

	return user, nil
}

func (api *API) logAuthEvent(ctx context.Context, params users.AuthLogParams) {
	if err := api.users.InsertAuthLog(ctx, params); err != nil && api.logger != nil {
		api.logger.ErrorContext(ctx, "failed to write auth log", "error", err, "event_type", params.EventType)
	}
}

type avatarUploadError struct {
	StatusCode int
	Code       string
	Message    string
}

func (err *avatarUploadError) Error() string {
	return err.Message
}

func (api *API) storeAvatarUpload(w http.ResponseWriter, r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarUploadBodySize)

	file, _, err := r.FormFile("avatar")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrMissingFile):
			return "", &avatarUploadError{StatusCode: http.StatusBadRequest, Code: "avatar_required", Message: "Avatar file is required."}
		case strings.Contains(err.Error(), "http: request body too large"):
			return "", &avatarUploadError{StatusCode: http.StatusRequestEntityTooLarge, Code: "avatar_too_large", Message: "Avatar file must be 5MB or smaller."}
		default:
			return "", fmt.Errorf("failed to read avatar upload: %w", err)
		}
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAvatarFileSizeBytes+1))
	if err != nil {
		return "", fmt.Errorf("failed to read avatar file: %w", err)
	}
	if len(data) == 0 {
		return "", &avatarUploadError{StatusCode: http.StatusBadRequest, Code: "avatar_required", Message: "Avatar file is required."}
	}
	if len(data) > maxAvatarFileSizeBytes {
		return "", &avatarUploadError{StatusCode: http.StatusRequestEntityTooLarge, Code: "avatar_too_large", Message: "Avatar file must be 5MB or smaller."}
	}

	contentType := http.DetectContentType(data[:min(512, len(data))])
	extension, allowed := avatarExtensionForContentType(contentType)
	if !allowed {
		return "", &avatarUploadError{StatusCode: http.StatusBadRequest, Code: "avatar_invalid_type", Message: "Avatar file must be a JPG, PNG, or WebP image."}
	}

	fileName, err := randomAvatarFileName(extension)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(users.AvatarDir(api.dataDir), fileName)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write avatar file: %w", err)
	}

	return fileName, nil
}

func avatarExtensionForContentType(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func randomAvatarFileName(extension string) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate avatar filename: %w", err)
	}

	return hex.EncodeToString(randomBytes) + extension, nil
}

func bearerTokenFromHeader(header string) (string, error) {
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

func readRefreshCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cookie.Value) == "" {
		return "", http.ErrNoCookie
	}

	return cookie.Value, nil
}

func setRefreshCookie(w http.ResponseWriter, r *http.Request, refreshToken string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     refreshCookiePath,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(r),
	})
}

func clearRefreshCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(r),
	})
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

func authRequestMetadata(r *http.Request) (string, string) {
	return clientAddr(r.RemoteAddr), strings.TrimSpace(r.UserAgent())
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

func parseRefreshToken(rawToken string) (string, string, error) {
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

func userAuthVersionCacheKey(userID int64) string {
	return fmt.Sprintf("user_auth_version:%d", userID)
}

func refreshSessionCacheKey(sessionID string) string {
	return "refresh_session:" + sessionID
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
