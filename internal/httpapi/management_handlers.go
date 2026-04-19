package httpapi

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"main/internal/audit"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/identity"
	"main/internal/systemsettings"
)

type systemSettingsResponse struct {
	AuthMode                          systemsettings.AuthMode         `json:"authMode"`
	RegistrationMode                  systemsettings.RegistrationMode `json:"registrationMode"`
	PasswordLoginEnabled              bool                            `json:"passwordLoginEnabled"`
	AdminUserCreateEnabled            bool                            `json:"adminUserCreateEnabled"`
	SelfServiceAccountDeletionEnabled bool                            `json:"selfServiceAccountDeletionEnabled"`
	UpdatedAt                         string                          `json:"updatedAt"`
}

type updateSystemSettingsRequest struct {
	AuthMode                          *systemsettings.AuthMode         `json:"authMode"`
	RegistrationMode                  *systemsettings.RegistrationMode `json:"registrationMode"`
	AdminUserCreateEnabled            *bool                            `json:"adminUserCreateEnabled"`
	SelfServiceAccountDeletionEnabled *bool                            `json:"selfServiceAccountDeletionEnabled"`
}

type managementUsersPageResponse struct {
	Items    []managedUserResponse `json:"items"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
	Total    int                   `json:"total"`
}

type managementUserUpsertRequest struct {
	Username string   `json:"username"`
	Email    *string  `json:"email"`
	Password string   `json:"password,omitempty"`
	RoleKeys []string `json:"roleKeys"`
}

func (api *API) getSystemSettingsHandler(w http.ResponseWriter, r *http.Request) {
	settings, err := api.settings.Get(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "system_settings_unavailable", "Failed to load system settings.")
		return
	}
	writeSuccessJSON(w, http.StatusOK, toSystemSettingsResponse(settings))
}

func (api *API) updateSystemSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var payload updateSystemSettingsRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}

	settings, err := api.settings.Update(r.Context(), systemsettings.UpdateInput{
		AuthMode:                          payload.AuthMode,
		RegistrationMode:                  payload.RegistrationMode,
		AdminUserCreateEnabled:            payload.AdminUserCreateEnabled,
		SelfServiceAccountDeletionEnabled: payload.SelfServiceAccountDeletionEnabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, systemsettings.ErrInvalidAuthMode):
			writeAPIError(w, http.StatusBadRequest, "invalid_auth_mode", "Invalid auth mode.")
		case errors.Is(err, systemsettings.ErrInvalidRegistrationMode):
			writeAPIError(w, http.StatusBadRequest, "invalid_registration_mode", "Invalid registration mode.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "system_settings_update_failed", "Failed to update system settings.")
		}
		return
	}

	writeSuccessJSON(w, http.StatusOK, toSystemSettingsResponse(settings))
}

func (api *API) listManagementUsersHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	result, err := api.identities.ListUsers(r.Context(), identity.ListUsersParams{
		Query:    r.URL.Query().Get("q"),
		RoleKey:  r.URL.Query().Get("role"),
		Status:   r.URL.Query().Get("status"),
		Page:     parsePositiveIntQuery(r, "page", 1),
		PageSize: parsePositiveIntQuery(r, "pageSize", 20),
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "management_users_unavailable", "Failed to load users.")
		return
	}

	actorTarget := authorization.UserTarget{
		UserID:   actor.User.ID,
		RoleKeys: actor.User.RoleKeys,
		Status:   actor.User.Status,
	}

	items := make([]managedUserResponse, 0, len(result.Items))
	for _, item := range result.Items {
		target := authorization.UserTarget{
			UserID:   item.ID,
			RoleKeys: item.RoleKeys,
			Status:   item.Status,
		}
		items = append(items, api.toManagedUserResponse(item, api.authorization.ManagedUserActions(actorTarget, target)))
	}

	writeSuccessJSON(w, http.StatusOK, managementUsersPageResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

func (api *API) createManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	var payload managementUserUpsertRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}
	if strings.TrimSpace(payload.Username) == "" {
		writeAPIError(w, http.StatusBadRequest, "username_required", "Username is required.")
		return
	}
	if len(strings.TrimSpace(payload.Password)) < auth.MinPasswordLength {
		writeAPIError(w, http.StatusBadRequest, "password_too_short", "Password must be at least 8 characters.")
		return
	}

	roleKey, allowedRoleKeys, ok := validateManagedRoleKeys(payload.RoleKeys, api.authorization.AllowedManagedRoleKeys(actor.User.RoleKeys, api.authorization.HasCapability(actor.Authorization.Capabilities, authorization.CapabilityManagementUsersCreate)))
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid_role_keys", "Invalid role assignment.")
		return
	}
	_ = allowedRoleKeys

	passwordHash, err := auth.HashPassword(payload.Password)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "user_create_failed", "Failed to create user.")
		return
	}

	ip, userAgent := requestMetadata(r)
	created, err := api.identities.CreateUser(r.Context(), identity.CreateUserParams{
		Username:         payload.Username,
		Email:            payload.Email,
		PasswordHash:     passwordHash,
		RoleKey:          roleKey,
		AssignedByUserID: new(actor.User.ID),
	}, identity.ActionAuditContext{
		ActorUserID:   new(actor.User.ID),
		AuthSessionID: new(actor.SessionID),
		IP:            nullableString(ip),
		UserAgent:     nullableString(userAgent),
	})
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrUsernameTaken):
			writeAPIError(w, http.StatusConflict, "username_taken", "Username is already in use.")
		case errors.Is(err, identity.ErrEmailTaken):
			writeAPIError(w, http.StatusConflict, "email_taken", "Email is already in use.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "user_create_failed", "Failed to create user.")
		}
		return
	}

	writeSuccessJSON(w, http.StatusCreated, api.toManagedUserResponse(created, nil))
}

func (api *API) updateManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	targetID, ok := parseInt64PathValue(w, r, "id")
	if !ok {
		return
	}

	targetUser, actorTarget, targetTarget, ok := api.loadManagedUserTargets(w, r, actor, targetID)
	if !ok {
		return
	}
	if !api.authorization.CanManageUser(actorTarget, targetTarget, authorization.UserActionUpdate) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "You do not have permission to update this user.")
		return
	}

	var payload managementUserUpsertRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}
	if strings.TrimSpace(payload.Username) == "" {
		writeAPIError(w, http.StatusBadRequest, "username_required", "Username is required.")
		return
	}

	roleKey, _, valid := validateManagedRoleKeys(payload.RoleKeys, api.authorization.AllowedManagedRoleKeys(actor.User.RoleKeys, true))
	if !valid {
		writeAPIError(w, http.StatusBadRequest, "invalid_role_keys", "Invalid role assignment.")
		return
	}

	ip, userAgent := requestMetadata(r)
	updated, err := api.identities.UpdateManagedUser(r.Context(), targetUser.ID, identity.UpdateManagedUserParams{
		Username: payload.Username,
		Email:    payload.Email,
		RoleKey:  roleKey,
	}, identity.ActionAuditContext{
		ActorUserID:   new(actor.User.ID),
		AuthSessionID: new(actor.SessionID),
		IP:            nullableString(ip),
		UserAgent:     nullableString(userAgent),
	})
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrUsernameTaken):
			writeAPIError(w, http.StatusConflict, "username_taken", "Username is already in use.")
		case errors.Is(err, identity.ErrEmailTaken):
			writeAPIError(w, http.StatusConflict, "email_taken", "Email is already in use.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "user_update_failed", "Failed to update user.")
		}
		return
	}

	writeSuccessJSON(w, http.StatusOK, api.toManagedUserResponse(updated, api.authorization.ManagedUserActions(actorTarget, authorization.UserTarget{
		UserID:   updated.ID,
		RoleKeys: updated.RoleKeys,
		Status:   updated.Status,
	})))
}

func (api *API) disableManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	api.setManagementUserStatusHandler(w, r, identity.StatusDisabled, authorization.UserActionDisable)
}

func (api *API) enableManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	api.setManagementUserStatusHandler(w, r, identity.StatusActive, authorization.UserActionEnable)
}

func (api *API) setManagementUserStatusHandler(w http.ResponseWriter, r *http.Request, status string, action authorization.UserAction) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	targetID, ok := parseInt64PathValue(w, r, "id")
	if !ok {
		return
	}

	targetUser, actorTarget, targetTarget, ok := api.loadManagedUserTargets(w, r, actor, targetID)
	if !ok {
		return
	}
	if !api.authorization.CanManageUser(actorTarget, targetTarget, action) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "You do not have permission to update this user.")
		return
	}

	ip, userAgent := requestMetadata(r)
	updated, err := api.identities.SetUserStatus(r.Context(), targetUser.ID, status, identity.ActionAuditContext{
		ActorUserID:   new(actor.User.ID),
		AuthSessionID: new(actor.SessionID),
		IP:            nullableString(ip),
		UserAgent:     nullableString(userAgent),
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "user_status_update_failed", "Failed to update user status.")
		return
	}

	writeSuccessJSON(w, http.StatusOK, api.toManagedUserResponse(updated, api.authorization.ManagedUserActions(actorTarget, authorization.UserTarget{
		UserID:   updated.ID,
		RoleKeys: updated.RoleKeys,
		Status:   updated.Status,
	})))
}

func (api *API) listAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	result, err := api.audit.List(r.Context(), audit.ListParams{
		Page:     parsePositiveIntQuery(r, "page", 1),
		PageSize: parsePositiveIntQuery(r, "pageSize", 50),
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_logs_unavailable", "Failed to load audit logs.")
		return
	}
	writeSuccessJSON(w, http.StatusOK, result)
}

func (api *API) loadManagedUserTargets(w http.ResponseWriter, r *http.Request, actor auth.Actor, targetID int64) (identity.UserWithRoles, authorization.UserTarget, authorization.UserTarget, bool) {
	targetUser, err := api.identities.GetUserByID(r.Context(), targetID)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			writeAPIError(w, http.StatusNotFound, "user_not_found", "User not found.")
		} else {
			writeAPIError(w, http.StatusInternalServerError, "user_unavailable", "Failed to load user.")
		}
		return identity.UserWithRoles{}, authorization.UserTarget{}, authorization.UserTarget{}, false
	}

	actorTarget := authorization.UserTarget{
		UserID:   actor.User.ID,
		RoleKeys: actor.User.RoleKeys,
		Status:   actor.User.Status,
	}
	targetTarget := authorization.UserTarget{
		UserID:   targetUser.ID,
		RoleKeys: targetUser.RoleKeys,
		Status:   targetUser.Status,
	}
	return targetUser, actorTarget, targetTarget, true
}

func toSystemSettingsResponse(settings systemsettings.Settings) systemSettingsResponse {
	return systemSettingsResponse{
		AuthMode:                          settings.AuthMode,
		RegistrationMode:                  settings.RegistrationMode,
		PasswordLoginEnabled:              settings.PasswordLoginEnabled,
		AdminUserCreateEnabled:            settings.AdminUserCreateEnabled,
		SelfServiceAccountDeletionEnabled: settings.SelfServiceAccountDeletionEnabled,
		UpdatedAt:                         settings.UpdatedAt.Format(time.RFC3339),
	}
}

func parsePositiveIntQuery(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseInt64PathValue(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	raw := strings.TrimSpace(r.PathValue(key))
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_id", "Invalid resource identifier.")
		return 0, false
	}
	return parsed, true
}

func validateManagedRoleKeys(roleKeys []string, allowedRoleKeys []string) (string, []string, bool) {
	if len(roleKeys) != 1 {
		return "", allowedRoleKeys, false
	}
	roleKey := strings.TrimSpace(roleKeys[0])
	if roleKey == "" {
		return "", allowedRoleKeys, false
	}

	if slices.Contains(allowedRoleKeys, roleKey) {
		return roleKey, allowedRoleKeys, true
	}
	return "", allowedRoleKeys, false
}
