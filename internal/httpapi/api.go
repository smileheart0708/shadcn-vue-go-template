package httpapi

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"main/internal/audit"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/identity"
	"main/internal/logging"
	"main/internal/setup"
	"main/internal/systemsettings"
)

type API struct {
	auth          *auth.Service
	authorization *authorization.Service
	identities    *identity.Service
	setup         *setup.Service
	settings      *systemsettings.Service
	audit         *audit.Service
	dataDir       string
	logger        *slog.Logger
	logStream     *logging.Stream
}

type viewerIdentityResponse struct {
	ID        int64   `json:"id"`
	Username  string  `json:"username"`
	Email     *string `json:"email"`
	AvatarURL *string `json:"avatarUrl"`
	Status    string  `json:"status"`
}

type viewerResponse struct {
	Identity      viewerIdentityResponse            `json:"identity"`
	Authorization authorization.ViewerAuthorization `json:"authorization"`
}

type sessionResponse struct {
	AccessToken string         `json:"accessToken"`
	TokenType   string         `json:"tokenType"`
	ExpiresAt   time.Time      `json:"expiresAt"`
	Viewer      viewerResponse `json:"viewer"`
}

type managedUserResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     *string   `json:"email"`
	AvatarURL *string   `json:"avatarUrl"`
	RoleKeys  []string  `json:"roleKeys"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Actions   []string  `json:"actions"`
}

type installStateResponse struct {
	SetupState     string     `json:"setupState"`
	SetupCompleted bool       `json:"setupCompleted"`
	OwnerUserID    *int64     `json:"ownerUserId"`
	CompletedAt    *time.Time `json:"completedAt"`
}

func (api *API) currentActor(ctx context.Context) (auth.Actor, bool) {
	return auth.ActorFromContext(ctx)
}

func (api *API) toViewerResponse(viewer auth.Viewer) viewerResponse {
	return viewerResponse{
		Identity: viewerIdentityResponse{
			ID:        viewer.User.ID,
			Username:  viewer.User.Username,
			Email:     viewer.User.Email,
			AvatarURL: api.toAvatarURL(viewer.User.AvatarPath),
			Status:    viewer.User.Status,
		},
		Authorization: viewer.Authorization,
	}
}

func (api *API) toSessionResponse(result auth.SessionResult) sessionResponse {
	return sessionResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   result.ExpiresAt,
		Viewer:      api.toViewerResponse(result.Viewer),
	}
}

func (api *API) toManagedUserResponse(user identity.UserWithRoles, actions []string) managedUserResponse {
	return managedUserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		AvatarURL: api.toAvatarURL(user.AvatarPath),
		RoleKeys:  cloneStringSlice(user.RoleKeys),
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Actions:   cloneStringSlice(actions),
	}
}

func (api *API) toAvatarURL(avatarPath *string) *string {
	if avatarPath == nil {
		return nil
	}
	filename := filepath.Base(strings.TrimSpace(*avatarPath))
	if filename == "" || filename == "." {
		return nil
	}
	url := "/api/avatars/" + filename
	return &url
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	return append([]string(nil), values...)
}
