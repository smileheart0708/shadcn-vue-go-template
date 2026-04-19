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
	"main/internal/authorization"
	"main/internal/identity"
)

const (
	maxAvatarFileSizeBytes  = 5 * 1024 * 1024
	maxAvatarUploadBodySize = maxAvatarFileSizeBytes + (1 << 20)
)

type updateProfileRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type deleteAccountResponse struct {
	Deleted bool `json:"deleted"`
}

func (api *API) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
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

	updated, err := api.identities.UpdateProfile(r.Context(), actor.User.ID, payload.Username, payload.Email)
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrUsernameTaken):
			writeAPIError(w, http.StatusConflict, "username_taken", "Username is already in use.")
		case errors.Is(err, identity.ErrEmailTaken):
			writeAPIError(w, http.StatusConflict, "email_taken", "Email is already in use.")
		case errors.Is(err, identity.ErrUserNotFound):
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "profile_update_failed", "Failed to update account profile.")
		}
		return
	}

	viewer, err := api.auth.BuildViewer(r.Context(), updated.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "profile_update_failed", "Failed to load updated viewer.")
		return
	}
	writeSuccessJSON(w, http.StatusOK, api.toViewerResponse(viewer))
}

func (api *API) updateAvatarHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	currentAvatarPath := ""
	if actor.User.AvatarPath != nil {
		currentAvatarPath = filepath.Join(identity.AvatarDir(api.dataDir), filepath.Base(*actor.User.AvatarPath))
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

	updated, err := api.identities.UpdateAvatar(r.Context(), actor.User.ID, &avatarFileName)
	if err != nil {
		_ = os.Remove(filepath.Join(identity.AvatarDir(api.dataDir), avatarFileName))
		writeAPIError(w, http.StatusInternalServerError, "avatar_update_failed", "Failed to update avatar.")
		return
	}

	if currentAvatarPath != "" {
		_ = os.Remove(currentAvatarPath)
	}

	viewer, err := api.auth.BuildViewer(r.Context(), updated.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "avatar_update_failed", "Failed to load updated viewer.")
		return
	}
	writeSuccessJSON(w, http.StatusOK, api.toViewerResponse(viewer))
}

func (api *API) changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	var payload changePasswordRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}

	ip, userAgent := requestMetadata(r)
	err := api.auth.ChangePassword(r.Context(), actor, payload.CurrentPassword, payload.NewPassword, ip, userAgent)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrCurrentPasswordInvalid):
			writeAPIError(w, http.StatusBadRequest, "current_password_invalid", "Current password is incorrect.")
		case errors.Is(err, auth.ErrPasswordTooShort):
			writeAPIError(w, http.StatusBadRequest, "password_too_short", "Password must be at least 8 characters.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "password_update_failed", "Failed to update password.")
		}
		return
	}

	api.auth.ClearRefreshCookie(w, r)
	writeSuccessJSON(w, http.StatusOK, passwordChangedResponse{PasswordChanged: true})
}

func (api *API) deleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}
	if !api.authorization.HasCapability(actor.Authorization.Capabilities, authorization.CapabilityAccountDeleteSelf) {
		writeAPIError(w, http.StatusForbidden, "account_delete_forbidden", "This account cannot delete itself.")
		return
	}

	currentAvatarPath := ""
	if actor.User.AvatarPath != nil {
		currentAvatarPath = filepath.Join(identity.AvatarDir(api.dataDir), filepath.Base(*actor.User.AvatarPath))
	}

	ip, userAgent := requestMetadata(r)
	if err := api.identities.DeleteUser(r.Context(), actor.User.ID, identity.ActionAuditContext{
		ActorUserID:   new(actor.User.ID),
		AuthSessionID: new(actor.SessionID),
		IP:            nullableString(ip),
		UserAgent:     nullableString(userAgent),
	}, "self_service"); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "account_delete_failed", "Failed to delete account.")
		return
	}

	if currentAvatarPath != "" {
		_ = os.Remove(currentAvatarPath)
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

	fullPath := filepath.Join(identity.AvatarDir(api.dataDir), filename)
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
			return "", fmt.Errorf("read avatar upload: %w", err)
		}
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAvatarFileSizeBytes+1))
	if err != nil {
		return "", fmt.Errorf("read avatar file: %w", err)
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

	if err := os.MkdirAll(identity.AvatarDir(api.dataDir), 0o755); err != nil {
		return "", fmt.Errorf("create avatar dir: %w", err)
	}

	fullPath := filepath.Join(identity.AvatarDir(api.dataDir), fileName)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write avatar file: %w", err)
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
		return "", fmt.Errorf("generate avatar filename: %w", err)
	}
	return hex.EncodeToString(randomBytes) + extension, nil
}
