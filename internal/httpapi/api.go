package httpapi

import (
	"context"
	"log/slog"

	"main/internal/auth"
	"main/internal/logging"
	"main/internal/users"
)

type API struct {
	users     *users.Store
	auth      *auth.Service
	dataDir   string
	logger    *slog.Logger
	logStream *logging.Stream
}

func NewAPI(userStore *users.Store, dataDir string, authOptions auth.Options) *API {
	return &API{
		users:   userStore,
		auth:    auth.NewService(authOptions, userStore),
		dataDir: dataDir,
	}
}

func (api *API) currentUser(ctx context.Context) (users.User, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
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
