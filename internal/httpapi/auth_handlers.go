package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"main/internal/auth"
	"main/internal/systemsettings"
	"main/internal/users"
)

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type registerRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email"`
	Password string  `json:"password"`
}

type registrationPolicyResponse struct {
	RegistrationMode systemsettings.RegistrationMode `json:"registrationMode"`
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
				EventType:     users.AuthLogEventLoginFailed,
				Success:       false,
				FailureReason: new("invalid_credentials"),
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
			UserID:        new(user.ID),
			Identifier:    nullableString(identifier),
			IP:            nullableString(requestIP),
			UserAgent:     nullableString(requestUserAgent),
			EventType:     users.AuthLogEventLoginFailed,
			Success:       false,
			FailureReason: new("invalid_credentials"),
		})
		writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Username/email or password is invalid.")
		return
	}
	if user.IsBanned {
		api.logAuthEvent(r.Context(), users.AuthLogParams{
			UserID:        new(user.ID),
			Identifier:    nullableString(identifier),
			IP:            nullableString(requestIP),
			UserAgent:     nullableString(requestUserAgent),
			EventType:     users.AuthLogEventLoginFailed,
			Success:       false,
			FailureReason: new("account_banned"),
		})
		writeAPIError(w, http.StatusForbidden, "account_banned", "This account has been banned.")
		return
	}

	response, err := api.issueLoginResponse(r.Context(), w, r, user, requestIP, requestUserAgent)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_create_failed", "Failed to create refresh session.")
		return
	}
	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:     new(user.ID),
		SessionID:  new(response.SessionID),
		Identifier: nullableString(identifier),
		IP:         nullableString(requestIP),
		UserAgent:  nullableString(requestUserAgent),
		EventType:  users.AuthLogEventLoginSuccess,
		Success:    true,
	})

	writeSuccessJSON(w, http.StatusOK, response.Payload)
}

func (api *API) registrationPolicyHandler(w http.ResponseWriter, r *http.Request) {
	settings, err := api.settings.Get(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "registration_policy_failed", "Failed to load registration policy.")
		return
	}

	writeSuccessJSON(w, http.StatusOK, registrationPolicyResponse{
		RegistrationMode: settings.RegistrationMode,
	})
}

func (api *API) registerHandler(w http.ResponseWriter, r *http.Request) {
	settings, err := api.settings.Get(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "registration_policy_failed", "Failed to load registration policy.")
		return
	}
	if settings.RegistrationMode != systemsettings.RegistrationModePassword {
		writeAPIError(w, http.StatusForbidden, "registration_disabled", "Registration is currently disabled.")
		return
	}

	var payload registerRequest
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

	passwordHash, err := users.HashPassword(payload.Password)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "registration_failed", "Failed to register user.")
		return
	}

	user, err := api.users.Create(r.Context(), users.CreateParams{
		Username:     payload.Username,
		Email:        payload.Email,
		PasswordHash: passwordHash,
		Role:         users.RoleUser,
	})
	if err != nil {
		switch {
		case errors.Is(err, users.ErrUsernameTaken):
			writeAPIError(w, http.StatusConflict, "username_taken", "Username is already in use.")
		case errors.Is(err, users.ErrEmailTaken):
			writeAPIError(w, http.StatusConflict, "email_taken", "Email is already in use.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "registration_failed", "Failed to register user.")
		}
		return
	}

	requestIP, requestUserAgent := requestMetadata(r)
	response, err := api.issueLoginResponse(r.Context(), w, r, user, requestIP, requestUserAgent)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_create_failed", "Failed to create refresh session.")
		return
	}

	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:     new(user.ID),
		SessionID:  new(response.SessionID),
		Identifier: nullableString(user.Username),
		IP:         nullableString(requestIP),
		UserAgent:  nullableString(requestUserAgent),
		EventType:  users.AuthLogEventRegisterSuccess,
		Success:    true,
	})

	writeSuccessJSON(w, http.StatusCreated, response.Payload)
}

func (api *API) refreshHandler(w http.ResponseWriter, r *http.Request) {
	rawToken, err := api.auth.ReadRefreshCookie(r)
	if err != nil {
		api.logAuthEvent(r.Context(), users.AuthLogParams{
			IP:            nullableString(clientAddr(r.RemoteAddr)),
			UserAgent:     nullableString(r.UserAgent()),
			EventType:     users.AuthLogEventRefreshFailed,
			Success:       false,
			FailureReason: new("refresh_cookie_missing"),
		})
		api.auth.ClearRefreshCookie(w, r)
		writeAPIError(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh session is invalid or expired.")
		return
	}

	result, attempt, err := api.auth.RefreshSession(r.Context(), rawToken)
	requestIP, requestUserAgent := requestMetadata(r)
	if err != nil {
		eventType := users.AuthLogEventRefreshFailed
		if errors.Is(err, auth.ErrTokenReuseDetected) {
			eventType = users.AuthLogEventTokenReuseDetected
		}
		api.logAuthEvent(r.Context(), users.AuthLogParams{
			UserID:        attempt.UserID,
			SessionID:     attempt.SessionID,
			IP:            nullableString(requestIP),
			UserAgent:     nullableString(requestUserAgent),
			EventType:     eventType,
			Success:       false,
			FailureReason: new(attempt.FailureReason),
		})
		api.auth.ClearRefreshCookie(w, r)
		writeAPIError(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh session is invalid or expired.")
		return
	}

	api.auth.SetRefreshCookie(w, r, result.RefreshToken)
	api.logAuthEvent(r.Context(), users.AuthLogParams{
		UserID:    new(result.User.ID),
		SessionID: new(result.SessionID),
		IP:        nullableString(requestIP),
		UserAgent: nullableString(requestUserAgent),
		EventType: users.AuthLogEventRefreshSuccess,
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
					UserID:    new(sessionState.UserID),
					SessionID: new(sessionID),
					IP:        nullableString(requestIP),
					UserAgent: nullableString(requestUserAgent),
					EventType: users.AuthLogEventLogout,
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

type issuedLoginResponse struct {
	SessionID string
	Payload   loginResponse
}

func (api *API) issueLoginResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, user users.User, requestIP string, requestUserAgent string) (issuedLoginResponse, error) {
	sessionID, refreshToken, err := api.auth.CreateRefreshSession(ctx, user, requestIP, requestUserAgent)
	if err != nil {
		return issuedLoginResponse{}, err
	}

	accessToken, expiresAt, err := api.auth.IssueToken(user, sessionID)
	if err != nil {
		_ = api.auth.RevokeSessionByID(ctx, sessionID, "access_token_issue_failed")
		return issuedLoginResponse{}, err
	}

	api.auth.SetRefreshCookie(w, r, refreshToken)

	return issuedLoginResponse{
		SessionID: sessionID,
		Payload: loginResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresAt:   expiresAt,
			User:        user.Public(),
		},
	}, nil
}
