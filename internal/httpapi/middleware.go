package httpapi

import (
	"net/http"

	"main/internal/auth"
)

type Middleware func(http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	wrapped := handler

	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}

	return wrapped
}

func RequireAuth(authService *auth.Service) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := auth.BearerTokenFromHeader(r.Header.Get("Authorization"))
			if err != nil {
				writeAPIError(w, http.StatusUnauthorized, "missing_token", "Missing bearer token.")
				return
			}

			principal, err := authService.ParseToken(token)
			if err != nil {
				writeAPIError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired token.")
				return
			}

			if err := authService.ValidatePrincipal(r.Context(), principal); err != nil {
				writeAPIError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired token.")
				return
			}

			ctx := auth.WithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(minRole int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := auth.PrincipalFromContext(r.Context())
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
