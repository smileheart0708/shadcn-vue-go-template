package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"main/internal/accountpolicies"
	"main/internal/audit"
	"main/internal/authorization"
	"main/internal/database"
	"main/internal/identity"
)

const (
	defaultIssuer         = "shadcn-vue-go-template"
	defaultAccessTTL      = 10 * time.Minute
	defaultRefreshIdleTTL = 7 * 24 * time.Hour
	defaultRefreshAbsTTL  = 30 * 24 * time.Hour
)

var (
	ErrMissingBearerToken     = errors.New("auth: missing bearer token")
	ErrInvalidToken           = errors.New("auth: invalid token")
	ErrInvalidCredentials     = errors.New("auth: invalid credentials")
	ErrAccountDisabled        = errors.New("auth: account disabled")
	ErrInvalidRefreshToken    = errors.New("auth: invalid refresh token")
	ErrRefreshTokenExpired    = errors.New("auth: refresh token expired")
	ErrRefreshTokenRevoked    = errors.New("auth: refresh token revoked")
	ErrTokenReuseDetected     = errors.New("auth: refresh token reuse detected")
	ErrRegistrationDisabled   = errors.New("auth: registration is disabled")
	ErrPasswordTooShort       = errors.New("auth: password too short")
	ErrCurrentPasswordInvalid = errors.New("auth: current password invalid")
)

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
	db                 *sql.DB
	identities         *identity.Service
	authorization      *authorization.Service
	policies           *accountpolicies.Service
	audit              *audit.Service
}

type Principal struct {
	UserID          int64
	SessionID       string
	SecurityVersion int64
}

type Actor struct {
	Principal
	User          identity.User
	Authorization authorization.ViewerAuthorization
}

type Viewer struct {
	User          identity.User
	Authorization authorization.ViewerAuthorization
}

type SessionResult struct {
	Viewer       Viewer
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
	SessionID    string
}

type RegisterParams struct {
	Username string
	Email    *string
	Password string
}

type refreshClaims struct {
	SessionID       string `json:"sid"`
	SecurityVersion int64  `json:"sv"`
	jwt.RegisteredClaims
}

type sessionRow struct {
	ID               string
	UserID           int64
	RefreshTokenHash string
	CreatedAt        time.Time
	LastUsedAt       time.Time
	LastRotatedAt    *time.Time
	ExpiresAt        time.Time
	IdleExpiresAt    time.Time
	RevokedAt        *time.Time
	RevokeReason     *string
}

func NewService(options Options, db *sql.DB, identities *identity.Service, authorizationService *authorization.Service, policies *accountpolicies.Service) *Service {
	if options.Issuer == "" {
		options.Issuer = defaultIssuer
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
		db:                 db,
		identities:         identities,
		authorization:      authorizationService,
		policies:           policies,
		audit:              audit.NewService(db),
	}
}

func (s *Service) AuthenticateRequest(ctx context.Context, header string) (Actor, error) {
	token, err := BearerTokenFromHeader(header)
	if err != nil {
		return Actor{}, err
	}

	principal, err := s.ParseAccessToken(token)
	if err != nil {
		return Actor{}, err
	}

	return s.actorForPrincipal(ctx, principal)
}

func (s *Service) BuildViewer(ctx context.Context, userID int64) (Viewer, error) {
	user, err := s.identities.GetUserByID(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}

	viewerAuthorization, err := s.buildViewerAuthorization(ctx, user.Role)
	if err != nil {
		return Viewer{}, err
	}

	return Viewer{
		User:          user,
		Authorization: viewerAuthorization,
	}, nil
}

func (s *Service) Login(ctx context.Context, identifier string, password string, ip string, userAgent string) (SessionResult, error) {
	record, err := s.identities.GetAuthRecordByIdentifier(ctx, identifier)
	if errors.Is(err, identity.ErrUserNotFound) {
		s.logAudit(ctx, audit.Entry{
			EventType: audit.EventLoginFailed,
			Outcome:   audit.OutcomeFailure,
			Reason:    new("invalid_credentials"),
			IP:        stringPointerOrNil(ip),
			UserAgent: stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return SessionResult{}, err
	}

	passwordMatches, err := VerifyPassword(password, record.PasswordHash)
	if err != nil {
		return SessionResult{}, fmt.Errorf("auth: verify password: %w", err)
	}
	if !passwordMatches {
		s.logAudit(ctx, audit.Entry{
			SubjectUserID: new(record.ID),
			EventType:     audit.EventLoginFailed,
			Outcome:       audit.OutcomeFailure,
			Reason:        new("invalid_credentials"),
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrInvalidCredentials
	}
	if record.Status != identity.StatusActive {
		s.logAudit(ctx, audit.Entry{
			SubjectUserID: new(record.ID),
			EventType:     audit.EventLoginFailed,
			Outcome:       audit.OutcomeFailure,
			Reason:        new("account_disabled"),
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrAccountDisabled
	}

	result, err := s.IssueSessionForUser(ctx, record.User, ip, userAgent)
	if err != nil {
		return SessionResult{}, err
	}

	s.logAudit(ctx, audit.Entry{
		ActorUserID:   new(record.ID),
		SubjectUserID: new(record.ID),
		AuthSessionID: new(result.SessionID),
		EventType:     audit.EventLoginSucceeded,
		Outcome:       audit.OutcomeSuccess,
		IP:            stringPointerOrNil(ip),
		UserAgent:     stringPointerOrNil(userAgent),
	})
	return result, nil
}

func (s *Service) Register(ctx context.Context, params RegisterParams, ip string, userAgent string) (SessionResult, error) {
	policies, err := s.policies.Get(ctx)
	if err != nil {
		return SessionResult{}, err
	}
	if !policies.PublicRegistrationEnabled {
		return SessionResult{}, ErrRegistrationDisabled
	}

	if len(strings.TrimSpace(params.Password)) < MinPasswordLength {
		return SessionResult{}, ErrPasswordTooShort
	}

	passwordHash, err := HashPassword(params.Password)
	if err != nil {
		return SessionResult{}, err
	}

	user, err := s.identities.CreateUser(ctx, identity.CreateUserParams{
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: passwordHash,
		Role:         authorization.RoleUser,
	}, identity.ActionAuditContext{})
	if err != nil {
		return SessionResult{}, err
	}

	result, err := s.IssueSessionForUser(ctx, user, ip, userAgent)
	if err != nil {
		return SessionResult{}, err
	}

	s.logAudit(ctx, audit.Entry{
		ActorUserID:   new(user.ID),
		SubjectUserID: new(user.ID),
		AuthSessionID: new(result.SessionID),
		EventType:     audit.EventRegistrationSucceeded,
		Outcome:       audit.OutcomeSuccess,
		IP:            stringPointerOrNil(ip),
		UserAgent:     stringPointerOrNil(userAgent),
	})
	return result, nil
}

func (s *Service) Refresh(ctx context.Context, rawToken string, ip string, userAgent string) (SessionResult, error) {
	sessionID, _, err := ParseRefreshToken(rawToken)
	if err != nil {
		s.logAudit(ctx, audit.Entry{
			EventType: audit.EventRefreshFailed,
			Outcome:   audit.OutcomeFailure,
			Reason:    new("invalid_refresh_token"),
			IP:        stringPointerOrNil(ip),
			UserAgent: stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrInvalidRefreshToken
	}

	session, err := s.getSessionByID(ctx, sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		s.logAudit(ctx, audit.Entry{
			AuthSessionID: new(sessionID),
			EventType:     audit.EventRefreshFailed,
			Outcome:       audit.OutcomeFailure,
			Reason:        new("session_not_found"),
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return SessionResult{}, err
	}

	expectedHash := hashRefreshToken(rawToken)
	if subtle.ConstantTimeCompare([]byte(session.RefreshTokenHash), []byte(expectedHash)) != 1 {
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "token_reuse_detected"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		s.logAudit(ctx, audit.Entry{
			SubjectUserID: new(session.UserID),
			AuthSessionID: new(session.ID),
			EventType:     audit.EventTokenReuseDetected,
			Outcome:       audit.OutcomeFailure,
			Reason:        new("token_reuse_detected"),
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrTokenReuseDetected
	}

	now := time.Now().UTC()
	switch {
	case session.RevokedAt != nil:
		s.logAudit(ctx, audit.Entry{
			SubjectUserID: new(session.UserID),
			AuthSessionID: new(session.ID),
			EventType:     audit.EventRefreshFailed,
			Outcome:       audit.OutcomeFailure,
			Reason:        new("session_revoked"),
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrRefreshTokenRevoked
	case now.After(session.ExpiresAt):
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "refresh_expired"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		s.logAudit(ctx, audit.Entry{
			SubjectUserID: new(session.UserID),
			AuthSessionID: new(session.ID),
			EventType:     audit.EventRefreshFailed,
			Outcome:       audit.OutcomeFailure,
			Reason:        new("refresh_expired"),
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrRefreshTokenExpired
	case now.After(session.IdleExpiresAt):
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "refresh_idle_expired"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		s.logAudit(ctx, audit.Entry{
			SubjectUserID: new(session.UserID),
			AuthSessionID: new(session.ID),
			EventType:     audit.EventRefreshFailed,
			Outcome:       audit.OutcomeFailure,
			Reason:        new("refresh_idle_expired"),
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrRefreshTokenExpired
	}

	user, err := s.identities.GetUserByID(ctx, session.UserID)
	if errors.Is(err, identity.ErrUserNotFound) {
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "user_not_found"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		s.logAudit(ctx, audit.Entry{
			SubjectUserID: new(session.UserID),
			AuthSessionID: new(session.ID),
			EventType:     audit.EventRefreshFailed,
			Outcome:       audit.OutcomeFailure,
			Reason:        new("user_not_found"),
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return SessionResult{}, err
	}
	if user.Status != identity.StatusActive {
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "account_disabled"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		s.logAudit(ctx, audit.Entry{
			SubjectUserID: new(session.UserID),
			AuthSessionID: new(session.ID),
			EventType:     audit.EventRefreshFailed,
			Outcome:       audit.OutcomeFailure,
			Reason:        new("account_disabled"),
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrAccountDisabled
	}

	nextSecret, err := randomTokenComponent(32)
	if err != nil {
		return SessionResult{}, err
	}
	nextRefreshToken := composeRefreshToken(session.ID, nextSecret)
	nextTokenHash := hashRefreshToken(nextRefreshToken)
	nextIdleExpiresAt := now.Add(s.refreshIdleTTL)

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE auth_sessions
		SET refresh_token_hash = ?,
			last_used_at = ?,
			last_rotated_at = ?,
			idle_expires_at = ?
		WHERE id = ? AND refresh_token_hash = ? AND revoked_at IS NULL`,
		nextTokenHash,
		now.Unix(),
		now.Unix(),
		nextIdleExpiresAt.Unix(),
		session.ID,
		expectedHash,
	)
	if err != nil {
		return SessionResult{}, fmt.Errorf("auth: rotate refresh session: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return SessionResult{}, fmt.Errorf("auth: rotate refresh session rows affected: %w", err)
	}
	if rowsAffected == 0 {
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "token_reuse_detected"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		s.logAudit(ctx, audit.Entry{
			SubjectUserID: new(session.UserID),
			AuthSessionID: new(session.ID),
			EventType:     audit.EventTokenReuseDetected,
			Outcome:       audit.OutcomeFailure,
			Reason:        new("token_reuse_detected"),
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
		return SessionResult{}, ErrTokenReuseDetected
	}

	accessToken, expiresAt, err := s.issueAccessToken(user, session.ID)
	if err != nil {
		_ = s.revokeSessionByID(ctx, session.ID, "access_token_issue_failed")
		return SessionResult{}, err
	}

	viewerAuthorization, err := s.buildViewerAuthorization(ctx, user.Role)
	if err != nil {
		return SessionResult{}, err
	}

	s.logAudit(ctx, audit.Entry{
		ActorUserID:   new(user.ID),
		SubjectUserID: new(user.ID),
		AuthSessionID: new(session.ID),
		EventType:     audit.EventRefreshSucceeded,
		Outcome:       audit.OutcomeSuccess,
		IP:            stringPointerOrNil(ip),
		UserAgent:     stringPointerOrNil(userAgent),
	})
	return SessionResult{
		Viewer: Viewer{
			User:          user,
			Authorization: viewerAuthorization,
		},
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: nextRefreshToken,
		SessionID:    session.ID,
	}, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string, ip string, userAgent string) error {
	sessionID, _, err := ParseRefreshToken(rawToken)
	if err != nil {
		return nil
	}

	session, err := s.getSessionByID(ctx, sessionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if err := s.revokeSessionByID(ctx, sessionID, "logout"); err != nil {
		return err
	}

	if err == nil && session.UserID > 0 {
		s.logAudit(ctx, audit.Entry{
			ActorUserID:   new(session.UserID),
			SubjectUserID: new(session.UserID),
			AuthSessionID: new(sessionID),
			EventType:     audit.EventLogoutSucceeded,
			Outcome:       audit.OutcomeSuccess,
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
		})
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, actor Actor, currentPassword string, newPassword string, ip string, userAgent string) error {
	if len(strings.TrimSpace(newPassword)) < MinPasswordLength {
		return ErrPasswordTooShort
	}

	record, err := s.identities.GetAuthRecordByID(ctx, actor.User.ID)
	if err != nil {
		return err
	}

	passwordMatches, err := VerifyPassword(currentPassword, record.PasswordHash)
	if err != nil {
		return fmt.Errorf("auth: verify current password: %w", err)
	}
	if !passwordMatches {
		return ErrCurrentPasswordInvalid
	}

	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE credentials
			SET password_hash = ?, password_changed_at = ?, updated_at = ?
			WHERE user_id = ?`,
			passwordHash,
			now.Unix(),
			now.Unix(),
			actor.User.ID,
		); err != nil {
			return fmt.Errorf("auth: update password credential: %w", err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`UPDATE users
			SET security_version = security_version + 1,
				updated_at = ?
			WHERE id = ?`,
			now.Unix(),
			actor.User.ID,
		); err != nil {
			return fmt.Errorf("auth: bump security version: %w", err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`UPDATE auth_sessions
			SET revoked_at = COALESCE(revoked_at, ?),
				revoke_reason = COALESCE(revoke_reason, 'password_changed')
			WHERE user_id = ? AND revoked_at IS NULL`,
			now.Unix(),
			actor.User.ID,
		); err != nil {
			return fmt.Errorf("auth: revoke sessions after password change: %w", err)
		}

		return audit.NewService(tx).Log(ctx, audit.Entry{
			ActorUserID:   new(actor.User.ID),
			SubjectUserID: new(actor.User.ID),
			AuthSessionID: new(actor.SessionID),
			EventType:     audit.EventPasswordChanged,
			Outcome:       audit.OutcomeSuccess,
			IP:            stringPointerOrNil(ip),
			UserAgent:     stringPointerOrNil(userAgent),
			OccurredAt:    now,
		})
	})
}

func (s *Service) IssueSessionForUser(ctx context.Context, user identity.User, ip string, userAgent string) (SessionResult, error) {
	if user.Status != identity.StatusActive {
		return SessionResult{}, ErrAccountDisabled
	}

	now := time.Now().UTC()
	sessionID, err := randomTokenComponent(16)
	if err != nil {
		return SessionResult{}, err
	}
	sessionSecret, err := randomTokenComponent(32)
	if err != nil {
		return SessionResult{}, err
	}

	refreshToken := composeRefreshToken(sessionID, sessionSecret)
	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO auth_sessions (
			id,
			user_id,
			refresh_token_hash,
			created_at,
			last_used_at,
			last_rotated_at,
			expires_at,
			idle_expires_at,
			revoked_at,
			revoke_reason,
			ip,
			user_agent
		) VALUES (?, ?, ?, ?, ?, NULL, ?, ?, NULL, NULL, ?, ?)`,
		sessionID,
		user.ID,
		hashRefreshToken(refreshToken),
		now.Unix(),
		now.Unix(),
		now.Add(s.refreshAbsoluteTTL).Unix(),
		now.Add(s.refreshIdleTTL).Unix(),
		stringPointerOrNil(ip),
		stringPointerOrNil(userAgent),
	); err != nil {
		return SessionResult{}, fmt.Errorf("auth: create refresh session: %w", err)
	}

	accessToken, expiresAt, err := s.issueAccessToken(user, sessionID)
	if err != nil {
		_ = s.revokeSessionByID(ctx, sessionID, "access_token_issue_failed")
		return SessionResult{}, err
	}

	viewerAuthorization, err := s.buildViewerAuthorization(ctx, user.Role)
	if err != nil {
		return SessionResult{}, err
	}

	return SessionResult{
		Viewer: Viewer{
			User:          user,
			Authorization: viewerAuthorization,
		},
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken,
		SessionID:    sessionID,
	}, nil
}

func (s *Service) ParseAccessToken(token string) (Principal, error) {
	parsed, err := jwt.ParseWithClaims(token, &refreshClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return Principal{}, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*refreshClaims)
	if !ok {
		return Principal{}, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 || claims.SecurityVersion <= 0 || strings.TrimSpace(claims.SessionID) == "" {
		return Principal{}, ErrInvalidToken
	}

	return Principal{
		UserID:          userID,
		SessionID:       claims.SessionID,
		SecurityVersion: claims.SecurityVersion,
	}, nil
}

func (s *Service) actorForPrincipal(ctx context.Context, principal Principal) (Actor, error) {
	user, err := s.identities.GetUserByID(ctx, principal.UserID)
	if err != nil {
		return Actor{}, ErrInvalidToken
	}
	if user.SecurityVersion != principal.SecurityVersion || user.Status != identity.StatusActive {
		return Actor{}, ErrInvalidToken
	}

	session, err := s.getSessionByID(ctx, principal.SessionID)
	if err != nil {
		return Actor{}, ErrInvalidToken
	}

	now := time.Now().UTC()
	switch {
	case session.UserID != principal.UserID:
		return Actor{}, ErrInvalidToken
	case session.RevokedAt != nil:
		return Actor{}, ErrInvalidToken
	case now.After(session.ExpiresAt):
		return Actor{}, ErrInvalidToken
	case now.After(session.IdleExpiresAt):
		return Actor{}, ErrInvalidToken
	}

	viewerAuthorization, err := s.buildViewerAuthorization(ctx, user.Role)
	if err != nil {
		return Actor{}, err
	}

	return Actor{
		Principal:     principal,
		User:          user,
		Authorization: viewerAuthorization,
	}, nil
}

func (s *Service) buildViewerAuthorization(ctx context.Context, role string) (authorization.ViewerAuthorization, error) {
	policies, err := s.policies.Get(ctx)
	if err != nil {
		return authorization.ViewerAuthorization{}, err
	}

	allowSelfDelete := policies.SelfServiceAccountDeletionEnabled && role != authorization.RoleOwner

	return s.authorization.ViewerAuthorization(role, authorization.ViewerOptions{
		AllowSelfDelete: allowSelfDelete,
	}), nil
}

func (s *Service) issueAccessToken(user identity.User, sessionID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims{
		SessionID:       sessionID,
		SecurityVersion: user.SecurityVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	encoded, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign access token: %w", err)
	}
	return encoded, expiresAt, nil
}

func (s *Service) getSessionByID(ctx context.Context, sessionID string) (sessionRow, error) {
	var session sessionRow
	var createdAt int64
	var lastUsedAt int64
	var lastRotatedAt sql.NullInt64
	var expiresAt int64
	var idleExpiresAt int64
	var revokedAt sql.NullInt64
	var revokeReason sql.NullString

	err := s.db.QueryRowContext(
		ctx,
		`SELECT
			id,
			user_id,
			refresh_token_hash,
			created_at,
			last_used_at,
			last_rotated_at,
			expires_at,
			idle_expires_at,
			revoked_at,
			revoke_reason
		FROM auth_sessions
		WHERE id = ?`,
		sessionID,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&createdAt,
		&lastUsedAt,
		&lastRotatedAt,
		&expiresAt,
		&idleExpiresAt,
		&revokedAt,
		&revokeReason,
	)
	if err != nil {
		return sessionRow{}, err
	}

	session.CreatedAt = time.Unix(createdAt, 0).UTC()
	session.LastUsedAt = time.Unix(lastUsedAt, 0).UTC()
	if lastRotatedAt.Valid {
		next := time.Unix(lastRotatedAt.Int64, 0).UTC()
		session.LastRotatedAt = &next
	}
	session.ExpiresAt = time.Unix(expiresAt, 0).UTC()
	session.IdleExpiresAt = time.Unix(idleExpiresAt, 0).UTC()
	if revokedAt.Valid {
		next := time.Unix(revokedAt.Int64, 0).UTC()
		session.RevokedAt = &next
	}
	if revokeReason.Valid {
		session.RevokeReason = &revokeReason.String
	}
	return session, nil
}

func (s *Service) revokeSessionByID(ctx context.Context, sessionID string, reason string) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, ?),
			revoke_reason = COALESCE(revoke_reason, ?)
		WHERE id = ?`,
		time.Now().UTC().Unix(),
		strings.TrimSpace(reason),
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("auth: revoke session: %w", err)
	}
	return nil
}

func (s *Service) logAudit(ctx context.Context, entry audit.Entry) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Log(ctx, entry)
}

func stringPointerOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return new(value)
}
