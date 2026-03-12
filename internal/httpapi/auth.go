package httpapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrMissingBearerToken = errors.New("missing bearer token")
var ErrInvalidToken = errors.New("invalid token")

type AuthUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type AuthOptions struct {
	Issuer   string
	Secret   []byte
	TTL      time.Duration
	User     AuthUser
	Password string
}

type AuthService struct {
	issuer       string
	secret       []byte
	ttl          time.Duration
	configured   AuthUser
	passwordHash []byte
}

type authClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	jwt.RegisteredClaims
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string    `json:"accessToken"`
	TokenType   string    `json:"tokenType"`
	ExpiresAt   time.Time `json:"expiresAt"`
	User        AuthUser  `json:"user"`
}

type meResponse struct {
	User AuthUser `json:"user"`
}

type authContextKey string

const authUserContextKey authContextKey = "auth-user"

func NewAuthService(options AuthOptions) *AuthService {
	if options.Issuer == "" {
		options.Issuer = "shadcn-vue-go-template"
	}
	if len(options.Secret) == 0 {
		options.Secret = []byte("change-me-in-production")
	}
	if options.TTL <= 0 {
		options.TTL = 24 * time.Hour
	}
	if options.User.Email == "" {
		options.User.Email = "admin@example.com"
	}
	if options.User.Name == "" {
		options.User.Name = "Administrator"
	}
	if options.User.ID == "" {
		options.User.ID = normalizeEmail(options.User.Email)
	}

	return &AuthService{
		issuer:       options.Issuer,
		secret:       options.Secret,
		ttl:          options.TTL,
		configured:   AuthUser{ID: options.User.ID, Email: normalizeEmail(options.User.Email), Name: options.User.Name},
		passwordHash: []byte(options.Password),
	}
}

func (s *AuthService) Authenticate(email string, password string) (loginResponse, error) {
	if !s.validCredentials(email, password) {
		return loginResponse{}, ErrInvalidCredentials
	}

	user := s.configured
	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)

	claims := authClaims{
		Email: user.Email,
		Name:  user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	encoded, err := token.SignedString(s.secret)
	if err != nil {
		return loginResponse{}, err
	}

	return loginResponse{
		AccessToken: encoded,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        user,
	}, nil
}

func (s *AuthService) ParseToken(token string) (AuthUser, error) {
	parsed, err := jwt.ParseWithClaims(token, &authClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidToken
		}

		return s.secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return AuthUser{}, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*authClaims)
	if !ok {
		return AuthUser{}, ErrInvalidToken
	}

	return AuthUser{
		ID:    claims.Subject,
		Email: normalizeEmail(claims.Email),
		Name:  claims.Name,
	}, nil
}

func RequireAuth(auth *AuthService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerTokenFromHeader(r.Header.Get("Authorization"))
			if err != nil {
				writeAPIError(w, http.StatusUnauthorized, "missing_token", "Missing bearer token.")
				return
			}

			user, err := auth.ParseToken(token)
			if err != nil {
				writeAPIError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired token.")
				return
			}

			ctx := context.WithValue(r.Context(), authUserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CurrentUser(ctx context.Context) (AuthUser, bool) {
	user, ok := ctx.Value(authUserContextKey).(AuthUser)
	return user, ok
}

func loginHandler(auth *AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload loginRequest
		if err := decodeJSON(w, r, &payload); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
			return
		}

		response, err := auth.Authenticate(payload.Email, payload.Password)
		if errors.Is(err, ErrInvalidCredentials) {
			writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Incorrect email or password.")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "token_issue_failed", "Failed to issue access token.")
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUser(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	writeJSON(w, http.StatusOK, meResponse{User: user})
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

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *AuthService) validCredentials(email string, password string) bool {
	emailBytes := []byte(normalizeEmail(email))
	passwordBytes := []byte(password)

	emailMatches := subtle.ConstantTimeCompare(emailBytes, []byte(s.configured.Email)) == 1
	passwordMatches := subtle.ConstantTimeCompare(passwordBytes, s.passwordHash) == 1

	return emailMatches && passwordMatches
}
