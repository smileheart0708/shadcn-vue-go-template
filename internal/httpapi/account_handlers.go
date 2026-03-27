package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"main/internal/auth"
	"main/internal/users"
)

const (
	minPasswordLength       = 8
	maxAvatarFileSizeBytes  = 5 * 1024 * 1024
	maxAvatarUploadBodySize = maxAvatarFileSizeBytes + (1 << 20)
)

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
	principal, _ := auth.PrincipalFromContext(r.Context())

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

	requestIP, requestUserAgent := requestMetadata(r)
	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:    new(user.ID),
		SessionID: new(principal.SessionID),
		IP:        nullableString(requestIP),
		UserAgent: nullableString(requestUserAgent),
		EventType: "password_changed",
		Success:   true,
	})
	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:        new(user.ID),
		SessionID:     nullableString(principal.SessionID),
		IP:            nullableString(requestIP),
		UserAgent:     nullableString(requestUserAgent),
		EventType:     "password_changed_forced_logout",
		Success:       true,
		FailureReason: new("all_sessions_revoked"),
	})

	if err := users.ClearBootstrapPasswordFile(api.dataDir); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "bootstrap_password_cleanup_failed", "Password was changed, but cleanup failed.")
		return
	}

	api.auth.ClearRefreshCookie(w, r)
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

	api.auth.ClearRefreshCookie(w, r)
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
