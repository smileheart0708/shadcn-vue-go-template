package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"main/internal/auth"
	"main/internal/users"
)

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

type logoutResponse struct {
	LoggedOut bool `json:"loggedOut"`
}

func (api *API) loginHandler(w http.ResponseWriter, r *http.Request) {
	var payload loginRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}

	identifier := strings.TrimSpace(payload.Identifier)
	requestIP, requestUserAgent := requestMetadata(r)

	user, err := api.users.FindByIdentifier(r.Context(), identifier)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			api.logAuthEvent(r.Context(), users.AuthLogParams{
				Identifier:    nullableString(identifier),
				IP:            nullableString(requestIP),
				UserAgent:     nullableString(requestUserAgent),
				EventType:     "login_failed",
				Success:       false,
				FailureReason: authStringPointer("invalid_credentials"),
			})
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
		api.logAuthEvent(r.Context(), users.AuthLogParams{
			UserID:        authInt64Pointer(user.ID),
			Identifier:    nullableString(identifier),
			IP:            nullableString(requestIP),
			UserAgent:     nullableString(requestUserAgent),
			EventType:     "login_failed",
			Success:       false,
			FailureReason: authStringPointer("invalid_credentials"),
		})
		writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Username/email or password is invalid.")
		return
	}

	sessionID, refreshToken, err := api.auth.CreateRefreshSession(r.Context(), user, requestIP, requestUserAgent)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_create_failed", "Failed to create refresh session.")
		return
	}

	accessToken, expiresAt, err := api.auth.IssueToken(user, sessionID)
	if err != nil {
		_ = api.auth.RevokeSessionByID(r.Context(), sessionID, "access_token_issue_failed")
		writeAPIError(w, http.StatusInternalServerError, "token_issue_failed", "Failed to issue access token.")
		return
	}

	api.auth.SetRefreshCookie(w, r, refreshToken)
	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:     authInt64Pointer(user.ID),
		SessionID:  authStringPointer(sessionID),
		Identifier: nullableString(identifier),
		IP:         nullableString(requestIP),
		UserAgent:  nullableString(requestUserAgent),
		EventType:  "login_success",
		Success:    true,
	})

	writeSuccessJSON(w, http.StatusOK, loginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        user.Public(),
	})
}

func (api *API) refreshHandler(w http.ResponseWriter, r *http.Request) {
	rawToken, err := api.auth.ReadRefreshCookie(r)
	if err != nil {
		api.logAuthEvent(r.Context(), users.AuthLogParams{
			IP:            nullableString(clientAddr(r.RemoteAddr)),
			UserAgent:     nullableString(r.UserAgent()),
			EventType:     "refresh_failed",
			Success:       false,
			FailureReason: authStringPointer("refresh_cookie_missing"),
		})
		api.auth.ClearRefreshCookie(w, r)
		writeAPIError(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh session is invalid or expired.")
		return
	}

	result, attempt, err := api.auth.RefreshSession(r.Context(), rawToken)
	requestIP, requestUserAgent := requestMetadata(r)
	if err != nil {
		eventType := "refresh_failed"
		if errors.Is(err, auth.ErrTokenReuseDetected) {
			eventType = "token_reuse_detected"
		}
		api.logAuthEvent(r.Context(), users.AuthLogParams{
			UserID:        attempt.UserID,
			SessionID:     attempt.SessionID,
			IP:            nullableString(requestIP),
			UserAgent:     nullableString(requestUserAgent),
			EventType:     eventType,
			Success:       false,
			FailureReason: authStringPointer(attempt.FailureReason),
		})
		api.auth.ClearRefreshCookie(w, r)
		writeAPIError(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh session is invalid or expired.")
		return
	}

	api.auth.SetRefreshCookie(w, r, result.RefreshToken)
	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:    authInt64Pointer(result.User.ID),
		SessionID: authStringPointer(result.SessionID),
		IP:        nullableString(requestIP),
		UserAgent: nullableString(requestUserAgent),
		EventType: "refresh_success",
		Success:   true,
	})

	writeSuccessJSON(w, http.StatusOK, loginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   result.ExpiresAt,
		User:        result.User.Public(),
	})
}

func (api *API) logoutHandler(w http.ResponseWriter, r *http.Request) {
	requestIP, requestUserAgent := requestMetadata(r)
	rawToken, err := api.auth.ReadRefreshCookie(r)
	if err == nil {
		sessionID, _, parseErr := auth.ParseRefreshToken(rawToken)
		if parseErr == nil {
			sessionState, stateErr := api.auth.LoadRefreshSessionState(r.Context(), sessionID)
			if stateErr == nil {
				api.logAuthEvent(r.Context(), users.AuthLogParams{
					UserID:    authInt64Pointer(sessionState.UserID),
					SessionID: authStringPointer(sessionID),
					IP:        nullableString(requestIP),
					UserAgent: nullableString(requestUserAgent),
					EventType: "logout",
					Success:   true,
				})
			}
		}

		_, _ = api.auth.RevokeRefreshSession(r.Context(), rawToken, "logout")
	}

	api.auth.ClearRefreshCookie(w, r)
	writeSuccessJSON(w, http.StatusOK, logoutResponse{LoggedOut: true})
}

func (api *API) meHandler(w http.ResponseWriter, r *http.Request) {
	user, err := api.currentUser(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	writeSuccessJSON(w, http.StatusOK, user.Public())
}
