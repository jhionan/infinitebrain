// Package middleware provides HTTP middleware and response helpers.
package middleware

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

// Response is the standard API response envelope.
type Response struct {
	Data  any    `json:"data"`
	Meta  *Meta          `json:"meta,omitempty"`
	Error *ErrorResponse `json:"error,omitempty"`
}

// Meta holds pagination and other metadata.
type Meta struct {
	Total      int64  `json:"total,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
}

// ErrorResponse is the standard error payload.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a JSON response with the given status code and data.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data}) //nolint:errcheck
}

// JSONWithMeta writes a JSON response with pagination metadata.
func JSONWithMeta(w http.ResponseWriter, status int, data any, meta *Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data, Meta: meta}) //nolint:errcheck
}

// JSONError writes a structured error response.
// It maps AppErrors to the correct HTTP status code.
func JSONError(w http.ResponseWriter, err error) {
	appErr := apperrors.AsAppError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)
	json.NewEncoder(w).Encode(Response{ //nolint:errcheck
		Error: &ErrorResponse{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}
