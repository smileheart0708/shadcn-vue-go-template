package httpapi

import (
	"context"
	"crypto/rand"
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

	"main/internal/logging"
	"main/internal/users"
)

const (
	minPasswordLength       = 8
	maxAvatarFileSizeBytes  = 5 * 1024 * 1024
	maxAvatarUploadBodySize = maxAvatarFileSizeBytes + (1 << 20)
)

var ErrMissingBearerToken = errors.New("missing bearer token")
var ErrInvalidToken = errors.New("invalid token")

type AuthOptions struct {
	Issuer string
	Secret []byte
	TTL    time.Duration
}

type API struct {
	users     *users.Store
	auth      *AuthService
	dataDir   string
	logger    *slog.Logger
	logStream *logging.Stream
}

type AuthService struct {
	issuer string
	secret []byte
	ttl    time.Duration
}

type AuthPrincipal struct {
	UserID int64
	Role   int
}

type authClaims struct {
	Role int `json:"role"`
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

const authUserContextKey authContextKey = "auth-user"

func NewAPI(db *sql.DB, dataDir string, authOptions AuthOptions) *API {
	return &API{
		users:   users.NewStore(db),
		auth:    NewAuthService(authOptions),
		dataDir: dataDir,
	}
}

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

	return &AuthService{
		issuer: options.Issuer,
		secret: options.Secret,
		ttl:    options.TTL,
	}
}

func (s *AuthService) IssueToken(user users.User) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)

	claims := authClaims{
		Role: user.Role,
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
	if err != nil || userID <= 0 {
		return AuthPrincipal{}, ErrInvalidToken
	}

	return AuthPrincipal{
		UserID: userID,
		Role:   claims.Role,
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

			principal, err := auth.ParseToken(token)
			if err != nil {
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

	user, err := api.users.FindByIdentifier(r.Context(), payload.Identifier)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
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
		writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Username/email or password is invalid.")
		return
	}

	accessToken, expiresAt, err := api.auth.IssueToken(user)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "token_issue_failed", "Failed to issue access token.")
		return
	}

	writeSuccessJSON(w, http.StatusOK, loginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        user.Public(),
	})
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

	if err := users.ClearBootstrapPasswordFile(api.dataDir); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "bootstrap_password_cleanup_failed", "Password was changed, but cleanup failed.")
		return
	}

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

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
