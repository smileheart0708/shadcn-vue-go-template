package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxJSONBodyBytes int64 = 1 << 20

type errorResponse struct {
	Success bool     `json:"success"`
	Error   apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type successResponse[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if payload == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func writeSuccessJSON[T any](w http.ResponseWriter, status int, data T) {
	writeJSON(w, status, successResponse[T]{
		Success: true,
		Data:    data,
	})
}

func writeAPIError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, errorResponse{
		Success: false,
		Error: apiError{
			Code:    code,
			Message: message,
		},
	})
}

func describeJSONDecodeError(err error) string {
	switch {
	case errors.Is(err, io.EOF):
		return "request body is required"
	case strings.Contains(err.Error(), "http: request body too large"):
		return fmt.Sprintf("request body must be at most %d bytes", maxJSONBodyBytes)
	default:
		return "invalid JSON request body"
	}
}
