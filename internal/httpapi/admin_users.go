package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"main/internal/users"
)

type adminUsersListResponse struct {
	Items    []users.AdminUser `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Total    int               `json:"total"`
}

type adminCreateUserRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email"`
	Password string  `json:"password"`
	Role     *int    `json:"role"`
}

type adminUpdateUserRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email"`
	Role     *int    `json:"role"`
}

func (api *API) adminListUsersHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := api.currentUser(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	params := users.ListParams{
		Query:    r.URL.Query().Get("q"),
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Sort:     strings.TrimSpace(r.URL.Query().Get("sort")),
		Page:     parsePositiveQueryInt(r, "page", 1),
		PageSize: parsePositiveQueryInt(r, "pageSize", 20),
	}

	if roleFilter := strings.TrimSpace(r.URL.Query().Get("role")); roleFilter != "" {
		role, parseErr := strconv.Atoi(roleFilter)
		if parseErr != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_role_filter", "Role filter is invalid.")
			return
		}
		params.Role = &role
	}

	if currentUser.Role == users.RoleAdmin {
		params.Role = new(users.RoleUser)
	}

	list, err := api.users.List(r.Context(), params)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "user_list_failed", "Failed to load users.")
		return
	}

	items := make([]users.AdminUser, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, item.Admin())
	}

	writeSuccessJSON(w, http.StatusOK, adminUsersListResponse{
		Items:    items,
		Page:     list.Page,
		PageSize: list.PageSize,
		Total:    list.Total,
	})
}

func (api *API) adminCreateUserHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := api.currentUser(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	var payload adminCreateUserRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}
	if strings.TrimSpace(payload.Username) == "" {
		writeAPIError(w, http.StatusBadRequest, "username_required", "Username is required.")
		return
	}
	if len(payload.Password) < minPasswordLength {
		writeAPIError(w, http.StatusBadRequest, "password_too_short", "Password must be at least 8 characters.")
		return
	}

	role := users.RoleUser
	if payload.Role != nil {
		role = *payload.Role
	}
	if !canAssignAdminManagedRole(currentUser.Role, role) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "You do not have permission to assign this role.")
		return
	}

	passwordHash, err := users.HashPassword(payload.Password)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "user_create_failed", "Failed to create user.")
		return
	}

	createdUser, err := api.users.AdminCreate(r.Context(), users.AdminCreateParams{
		Username:     payload.Username,
		Email:        payload.Email,
		PasswordHash: passwordHash,
		Role:         role,
	})
	if err != nil {
		switch {
		case errors.Is(err, users.ErrUsernameTaken):
			writeAPIError(w, http.StatusConflict, "username_taken", "Username is already in use.")
		case errors.Is(err, users.ErrEmailTaken):
			writeAPIError(w, http.StatusConflict, "email_taken", "Email is already in use.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "user_create_failed", "Failed to create user.")
		}
		return
	}

	writeSuccessJSON(w, http.StatusCreated, createdUser.Admin())
}

func (api *API) adminUpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, targetUser, targetID, err := api.loadAdminManagedTargetUser(r)
	if err != nil {
		writeAdminTargetUserError(w, err)
		return
	}

	var payload adminUpdateUserRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}
	if strings.TrimSpace(payload.Username) == "" {
		writeAPIError(w, http.StatusBadRequest, "username_required", "Username is required.")
		return
	}

	role := targetUser.Role
	if payload.Role != nil {
		role = *payload.Role
	}
	if targetID == currentUser.ID && role != targetUser.Role {
		writeAPIError(w, http.StatusForbidden, "self_role_change_forbidden", "You cannot change your own role from the admin user management page.")
		return
	}
	if !canAssignAdminManagedRole(currentUser.Role, role) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "You do not have permission to assign this role.")
		return
	}

	updatedUser, err := api.users.AdminUpdate(r.Context(), targetID, users.AdminUpdateParams{
		Username: payload.Username,
		Email:    payload.Email,
		Role:     role,
	})
	if err != nil {
		switch {
		case errors.Is(err, users.ErrUsernameTaken):
			writeAPIError(w, http.StatusConflict, "username_taken", "Username is already in use.")
		case errors.Is(err, users.ErrEmailTaken):
			writeAPIError(w, http.StatusConflict, "email_taken", "Email is already in use.")
		case errors.Is(err, users.ErrUserNotFound):
			writeAPIError(w, http.StatusNotFound, "user_not_found", "User was not found.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "user_update_failed", "Failed to update user.")
		}
		return
	}

	if updatedUser.Role != targetUser.Role {
		api.auth.InvalidateUser(updatedUser.ID)
	}

	writeSuccessJSON(w, http.StatusOK, updatedUser.Admin())
}

func (api *API) adminBanUserHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, _, targetID, err := api.loadAdminManagedTargetUser(r)
	if err != nil {
		writeAdminTargetUserError(w, err)
		return
	}
	if targetID == currentUser.ID {
		writeAPIError(w, http.StatusForbidden, "self_ban_forbidden", "You cannot ban your own account.")
		return
	}

	bannedUser, err := api.users.Ban(r.Context(), targetID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			writeAPIError(w, http.StatusNotFound, "user_not_found", "User was not found.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "user_ban_failed", "Failed to ban user.")
		return
	}
	if err := api.auth.RevokeRefreshSessionsByUser(r.Context(), targetID, "account_banned"); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "user_ban_failed", "Failed to revoke active sessions for banned user.")
		return
	}
	api.auth.InvalidateUser(targetID)

	writeSuccessJSON(w, http.StatusOK, bannedUser.Admin())
}

func (api *API) adminUnbanUserHandler(w http.ResponseWriter, r *http.Request) {
	_, _, targetID, err := api.loadAdminManagedTargetUser(r)
	if err != nil {
		writeAdminTargetUserError(w, err)
		return
	}

	unbannedUser, err := api.users.Unban(r.Context(), targetID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			writeAPIError(w, http.StatusNotFound, "user_not_found", "User was not found.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "user_unban_failed", "Failed to unban user.")
		return
	}
	api.auth.InvalidateUser(targetID)

	writeSuccessJSON(w, http.StatusOK, unbannedUser.Admin())
}

func (api *API) loadAdminManagedTargetUser(r *http.Request) (users.User, users.User, int64, error) {
	currentUser, err := api.currentUser(r.Context())
	if err != nil {
		return users.User{}, users.User{}, 0, err
	}

	targetID, err := parsePathInt64(r, "id")
	if err != nil {
		return users.User{}, users.User{}, 0, err
	}

	targetUser, err := api.users.GetByID(r.Context(), targetID)
	if err != nil {
		return users.User{}, users.User{}, targetID, err
	}
	if !canManageAdminTarget(currentUser.Role, targetUser.Role) {
		return users.User{}, users.User{}, targetID, errAdminTargetForbidden
	}

	return currentUser, targetUser, targetID, nil
}

var errAdminTargetForbidden = errors.New("admin target forbidden")

func writeAdminTargetUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, users.ErrUserNotFound):
		writeAPIError(w, http.StatusNotFound, "user_not_found", "User was not found.")
	case errors.Is(err, errAdminTargetForbidden):
		writeAPIError(w, http.StatusForbidden, "forbidden", "You do not have permission to manage this user.")
	default:
		var statusErr *pathParamError
		if errors.As(err, &statusErr) {
			writeAPIError(w, http.StatusBadRequest, statusErr.Code, statusErr.Message)
			return
		}
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
	}
}

type pathParamError struct {
	Code    string
	Message string
}

func (e *pathParamError) Error() string {
	return e.Message
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	value := strings.TrimSpace(r.PathValue(key))
	if value == "" {
		return 0, &pathParamError{Code: "invalid_user_id", Message: "User id is required."}
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, &pathParamError{Code: "invalid_user_id", Message: "User id is invalid."}
	}

	return parsed, nil
}

func parsePositiveQueryInt(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}

func canManageAdminTarget(actorRole int, targetRole int) bool {
	switch actorRole {
	case users.RoleSuperAdmin:
		return targetRole == users.RoleUser || targetRole == users.RoleAdmin
	case users.RoleAdmin:
		return targetRole == users.RoleUser
	default:
		return false
	}
}

func canAssignAdminManagedRole(actorRole int, role int) bool {
	switch actorRole {
	case users.RoleSuperAdmin:
		return role == users.RoleUser || role == users.RoleAdmin
	case users.RoleAdmin:
		return role == users.RoleUser
	default:
		return false
	}
}
