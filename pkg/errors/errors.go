// Package apperrors defines typed application errors for consistent HTTP responses.
package apperrors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is a typed error with an HTTP status code and client-facing code.
type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Is reports whether this error matches target by comparing error codes.
// This allows errors.Is(apperrors.ErrNotFound.Wrap(err), apperrors.ErrNotFound) to return true.
func (e *AppError) Is(target error) bool {
	var t *AppError
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// Wrap returns a new AppError wrapping an underlying error with additional context.
func (e *AppError) Wrap(err error) *AppError {
	return &AppError{
		HTTPStatus: e.HTTPStatus,
		Code:       e.Code,
		Message:    e.Message,
		Err:        err,
	}
}

// WithMessage returns a copy of the AppError with a new message.
func (e *AppError) WithMessage(msg string) *AppError {
	return &AppError{
		HTTPStatus: e.HTTPStatus,
		Code:       e.Code,
		Message:    msg,
		Err:        e.Err,
	}
}

// Sentinel errors — use these throughout the application.
var (
	ErrNotFound = &AppError{
		HTTPStatus: http.StatusNotFound,
		Code:       "NOT_FOUND",
		Message:    "resource not found",
	}
	ErrUnauthorized = &AppError{
		HTTPStatus: http.StatusUnauthorized,
		Code:       "UNAUTHORIZED",
		Message:    "authentication required",
	}
	ErrForbidden = &AppError{
		HTTPStatus: http.StatusForbidden,
		Code:       "FORBIDDEN",
		Message:    "access denied",
	}
	ErrValidation = &AppError{
		HTTPStatus: http.StatusUnprocessableEntity,
		Code:       "VALIDATION_ERROR",
		Message:    "validation failed",
	}
	ErrConflict = &AppError{
		HTTPStatus: http.StatusConflict,
		Code:       "CONFLICT",
		Message:    "resource already exists",
	}
	ErrPlanLimitReached = &AppError{
		HTTPStatus: http.StatusPaymentRequired,
		Code:       "PLAN_LIMIT_REACHED",
		Message:    "plan limit reached",
	}
	ErrInternal = &AppError{
		HTTPStatus: http.StatusInternalServerError,
		Code:       "INTERNAL_ERROR",
		Message:    "an unexpected error occurred",
	}
	ErrBadRequest = &AppError{
		HTTPStatus: http.StatusBadRequest,
		Code:       "BAD_REQUEST",
		Message:    "invalid request",
	}
)

// IsNotFound returns true if the error is or wraps ErrNotFound.
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == ErrNotFound.Code
	}
	return false
}

// IsUnauthorized returns true if the error is or wraps ErrUnauthorized.
func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == ErrUnauthorized.Code
	}
	return false
}

// AsAppError extracts an AppError from an error chain.
// Returns ErrInternal if the error is not an AppError.
func AsAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return ErrInternal.Wrap(err)
}
