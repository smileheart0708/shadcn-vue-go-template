package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickmn/go-cache"

	"main/internal/users"
)

const (
	defaultCacheTTL       = 5 * time.Minute
	sessionCacheTTL       = 2 * time.Minute
	defaultIssuer         = "shadcn-vue-go-template"
	defaultSigningSecret  = "change-me-in-production"
	defaultAccessTTL      = 10 * time.Minute
	defaultRefreshIdleTTL = 7 * 24 * time.Hour
	defaultRefreshAbsTTL  = 30 * 24 * time.Hour
)

var ErrMissingBearerToken = errors.New("missing bearer token")
var ErrInvalidToken = errors.New("invalid token")
var ErrInvalidRefreshToken = errors.New("invalid refresh token")
var ErrRefreshTokenExpired = errors.New("refresh token expired")
var ErrRefreshTokenRevoked = errors.New("refresh token revoked")
var ErrTokenReuseDetected = errors.New("refresh token reuse detected")

type Options struct {
	Issuer             string
	Secret             []byte
	TTL                time.Duration
	RefreshIdleTTL     time.Duration
	RefreshAbsoluteTTL time.Duration
	RefreshCookieName  string
}

type Service struct {
	issuer             string
	secret             []byte
	ttl                time.Duration
	refreshIdleTTL     time.Duration
	refreshAbsoluteTTL time.Duration
	refreshCookieName  string
	users              *users.Store
	authVersionCache   *cache.Cache
	sessionCache       *cache.Cache
}

type Principal struct {
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

type RefreshResult struct {
	User         users.User
	SessionID    string
	RefreshToken string
	AccessToken  string
	ExpiresAt    time.Time
}

type RefreshAttempt struct {
	UserID        *int64
	SessionID     *string
	FailureReason string
}

type RefreshSessionState struct {
	UserID        int64
	TokenHash     string
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
	RevokedAt     *time.Time
}

func NewService(options Options, userStore *users.Store) *Service {
	if options.Issuer == "" {
		options.Issuer = defaultIssuer
	}
	if len(options.Secret) == 0 {
		options.Secret = []byte(defaultSigningSecret)
	}
	if options.TTL <= 0 {
		options.TTL = defaultAccessTTL
	}
	if options.RefreshIdleTTL <= 0 {
		options.RefreshIdleTTL = defaultRefreshIdleTTL
	}
	if options.RefreshAbsoluteTTL <= 0 {
		options.RefreshAbsoluteTTL = defaultRefreshAbsTTL
	}

	return &Service{
		issuer:             options.Issuer,
		secret:             options.Secret,
		ttl:                options.TTL,
		refreshIdleTTL:     options.RefreshIdleTTL,
		refreshAbsoluteTTL: options.RefreshAbsoluteTTL,
		refreshCookieName:  ResolveRefreshCookieName(options),
		users:              userStore,
		authVersionCache:   cache.New(defaultCacheTTL, defaultCacheTTL),
		sessionCache:       cache.New(sessionCacheTTL, sessionCacheTTL),
	}
}

func (s *Service) IssueToken(user users.User, sessionID string) (string, time.Time, error) {
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

func (s *Service) ParseToken(token string) (Principal, error) {
	parsed, err := jwt.ParseWithClaims(token, &authClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidToken
		}

		return s.secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return Principal{}, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*authClaims)
	if !ok {
		return Principal{}, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 || claims.AuthVersion <= 0 || strings.TrimSpace(claims.SessionID) == "" {
		return Principal{}, ErrInvalidToken
	}

	return Principal{
		UserID:      userID,
		SessionID:   claims.SessionID,
		Role:        claims.Role,
		AuthVersion: claims.AuthVersion,
	}, nil
}

func (s *Service) ValidatePrincipal(ctx context.Context, principal Principal) error {
	authVersion, err := s.loadAuthVersion(ctx, principal.UserID)
	if err != nil {
		return err
	}
	if authVersion != principal.AuthVersion {
		return ErrInvalidToken
	}

	sessionState, err := s.loadRefreshSessionState(ctx, principal.SessionID)
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

func (s *Service) CreateRefreshSession(ctx context.Context, user users.User, ip string, userAgent string) (string, string, error) {
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
		IP:            stringPtrOrNil(ip),
		UserAgent:     stringPtrOrNil(userAgent),
	}

	if err := s.users.CreateRefreshSession(ctx, session); err != nil {
		return "", "", err
	}

	s.setRefreshSessionState(session.ID, RefreshSessionState{
		UserID:        session.UserID,
		TokenHash:     session.TokenHash,
		ExpiresAt:     session.ExpiresAt,
		IdleExpiresAt: session.IdleExpiresAt,
	})

	return session.ID, refreshToken, nil
}

func (s *Service) RefreshSession(ctx context.Context, rawToken string) (RefreshResult, RefreshAttempt, error) {
	sessionID, _, err := ParseRefreshToken(rawToken)
	if err != nil {
		return RefreshResult{}, RefreshAttempt{FailureReason: "refresh_token_invalid"}, ErrInvalidRefreshToken
	}

	attempt := RefreshAttempt{
		SessionID:     new(sessionID),
		FailureReason: "refresh_failed",
	}

	sessionState, err := s.loadRefreshSessionState(ctx, sessionID)
	if err != nil {
		if errors.Is(err, users.ErrRefreshSessionNotFound) {
			attempt.FailureReason = "refresh_session_not_found"
			return RefreshResult{}, attempt, ErrInvalidRefreshToken
		}
		return RefreshResult{}, attempt, err
	}

	attempt.UserID = new(sessionState.UserID)

	tokenHash := hashRefreshToken(rawToken)
	if subtle.ConstantTimeCompare([]byte(sessionState.TokenHash), []byte(tokenHash)) != 1 {
		attempt.FailureReason = "token_reuse_detected"
		_ = s.revokeRefreshSession(ctx, sessionID, "token_reuse_detected")
		return RefreshResult{}, attempt, ErrTokenReuseDetected
	}

	now := time.Now().UTC()
	switch {
	case sessionState.RevokedAt != nil:
		attempt.FailureReason = "refresh_session_revoked"
		return RefreshResult{}, attempt, ErrRefreshTokenRevoked
	case now.After(sessionState.ExpiresAt):
		attempt.FailureReason = "refresh_token_expired"
		_ = s.revokeRefreshSession(ctx, sessionID, "refresh_token_expired")
		return RefreshResult{}, attempt, ErrRefreshTokenExpired
	case now.After(sessionState.IdleExpiresAt):
		attempt.FailureReason = "refresh_token_idle_expired"
		_ = s.revokeRefreshSession(ctx, sessionID, "refresh_token_idle_expired")
		return RefreshResult{}, attempt, ErrRefreshTokenExpired
	}

	user, err := s.users.GetByID(ctx, sessionState.UserID)
	if err != nil {
		attempt.FailureReason = "user_not_found"
		_ = s.revokeRefreshSession(ctx, sessionID, "user_not_found")
		return RefreshResult{}, attempt, ErrInvalidRefreshToken
	}

	authVersion, err := s.loadAuthVersion(ctx, user.ID)
	if err != nil {
		return RefreshResult{}, attempt, err
	}
	user.AuthVersion = authVersion

	nextSecret, err := randomTokenComponent(32)
	if err != nil {
		return RefreshResult{}, attempt, err
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
		return RefreshResult{}, attempt, err
	}

	s.setRefreshSessionState(sessionID, RefreshSessionState{
		UserID:        user.ID,
		TokenHash:     nextTokenHash,
		ExpiresAt:     sessionState.ExpiresAt,
		IdleExpiresAt: nextIdleExpiry,
	})

	accessToken, expiresAt, err := s.IssueToken(user, sessionID)
	if err != nil {
		return RefreshResult{}, attempt, err
	}

	return RefreshResult{
		User:         user,
		SessionID:    sessionID,
		RefreshToken: nextRefreshToken,
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
	}, attempt, nil
}

func (s *Service) RevokeRefreshSession(ctx context.Context, rawToken string, reason string) (*string, error) {
	sessionID, _, err := ParseRefreshToken(rawToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if err := s.revokeRefreshSession(ctx, sessionID, reason); err != nil {
		return new(sessionID), err
	}

	return new(sessionID), nil
}

func (s *Service) RevokeRefreshSessionsByUser(ctx context.Context, userID int64, reason string) error {
	if _, err := s.users.RevokeRefreshSessionsByUser(ctx, userID, reason, time.Now().UTC()); err != nil {
		return err
	}

	s.sessionCache.Flush()
	return nil
}

func (s *Service) InvalidateUser(userID int64) {
	s.authVersionCache.Delete(userAuthVersionCacheKey(userID))
}

func (s *Service) RevokeSessionByID(ctx context.Context, sessionID string, reason string) error {
	return s.revokeRefreshSession(ctx, sessionID, reason)
}

func (s *Service) LoadRefreshSessionState(ctx context.Context, sessionID string) (RefreshSessionState, error) {
	return s.loadRefreshSessionState(ctx, sessionID)
}

func (s *Service) loadAuthVersion(ctx context.Context, userID int64) (int64, error) {
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

func (s *Service) loadRefreshSessionState(ctx context.Context, sessionID string) (RefreshSessionState, error) {
	cacheKey := refreshSessionCacheKey(sessionID)
	if cached, ok := s.sessionCache.Get(cacheKey); ok {
		if state, ok := cached.(RefreshSessionState); ok {
			return state, nil
		}
	}

	session, err := s.users.GetRefreshSessionByID(ctx, sessionID)
	if err != nil {
		return RefreshSessionState{}, err
	}

	state := RefreshSessionState{
		UserID:        session.UserID,
		TokenHash:     session.TokenHash,
		ExpiresAt:     session.ExpiresAt,
		IdleExpiresAt: session.IdleExpiresAt,
		RevokedAt:     session.RevokedAt,
	}
	s.setRefreshSessionState(sessionID, state)
	return state, nil
}

func (s *Service) setRefreshSessionState(sessionID string, state RefreshSessionState) {
	s.sessionCache.Set(refreshSessionCacheKey(sessionID), state, cache.DefaultExpiration)
}

func (s *Service) revokeRefreshSession(ctx context.Context, sessionID string, reason string) error {
	err := s.users.RevokeRefreshSession(ctx, sessionID, reason, time.Now().UTC())
	if err != nil && !errors.Is(err, users.ErrRefreshSessionNotFound) {
		return err
	}

	s.sessionCache.Delete(refreshSessionCacheKey(sessionID))
	return nil
}
