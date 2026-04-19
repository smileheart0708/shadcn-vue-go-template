package httpapi

import (
	"net/http"
	"slices"

	"main/internal/auth"
	"main/internal/setup"
)

type Middleware func(http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped
}

func RequireSetupCompleted(setupService *setup.Service) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			state, err := setupService.GetState(r.Context())
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "setup_state_unavailable", "Failed to load install state.")
				return
			}
			if !state.SetupCompleted {
				writeAPIError(w, http.StatusServiceUnavailable, "setup_required", "The application must be initialized before this endpoint can be used.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAuth(authService *auth.Service) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, err := authService.AuthenticateRequest(r.Context(), r.Header.Get("Authorization"))
			if err != nil {
				switch err {
				case auth.ErrMissingBearerToken:
					writeAPIError(w, http.StatusUnauthorized, "missing_token", "Missing bearer token.")
				default:
					writeAPIError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired token.")
				}
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithActor(r.Context(), actor)))
		})
	}
}

func RequireCapability(capability string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := auth.ActorFromContext(r.Context())
			if !ok {
				writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
				return
			}

			if slices.Contains(actor.Authorization.Capabilities, capability) {
				next.ServeHTTP(w, r)
				return
			}

			writeAPIError(w, http.StatusForbidden, "forbidden", "You do not have permission to access this resource.")
		})
	}
}
