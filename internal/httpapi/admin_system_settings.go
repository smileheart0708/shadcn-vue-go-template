package httpapi

import (
	"errors"
	"net/http"
	"time"

	"main/internal/systemsettings"
)

type adminSystemSettingsResponse struct {
	RegistrationMode systemsettings.RegistrationMode `json:"registrationMode"`
	UpdatedAt        string                          `json:"updatedAt"`
}

type updateRegistrationSettingsRequest struct {
	RegistrationMode systemsettings.RegistrationMode `json:"registrationMode"`
}

func (api *API) adminGetSystemSettingsHandler(w http.ResponseWriter, r *http.Request) {
	settings, err := api.settings.Get(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "system_settings_load_failed", "Failed to load system settings.")
		return
	}

	writeSuccessJSON(w, http.StatusOK, newAdminSystemSettingsResponse(settings))
}

func (api *API) adminUpdateRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	var payload updateRegistrationSettingsRequest
	if err := decodeJSON(w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}

	settings, err := api.settings.UpdateRegistrationMode(r.Context(), payload.RegistrationMode)
	if err != nil {
		switch {
		case errors.Is(err, systemsettings.ErrInvalidRegistrationMode):
			writeAPIError(w, http.StatusBadRequest, "invalid_registration_mode", "Registration mode is invalid.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "system_settings_update_failed", "Failed to update system settings.")
		}
		return
	}

	writeSuccessJSON(w, http.StatusOK, newAdminSystemSettingsResponse(settings))
}

func newAdminSystemSettingsResponse(settings systemsettings.Settings) adminSystemSettingsResponse {
	return adminSystemSettingsResponse{
		RegistrationMode: settings.RegistrationMode,
		UpdatedAt:        settings.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
