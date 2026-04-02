package apperrors_test

import (
	"fmt"
	"net/http"
	"testing"

	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

func TestAppError_Error_IncludesMessageAndWrappedError(t *testing.T) {
	underlying := fmt.Errorf("db connection refused")
	err := apperrors.ErrNotFound.Wrap(underlying)

	got := err.Error()
	if got != "resource not found: db connection refused" {
		t.Errorf("unexpected error string: %q", got)
	}
}

func TestAppError_Error_WithoutWrappedError(t *testing.T) {
	err := apperrors.ErrNotFound
	if err.Error() != "resource not found" {
		t.Errorf("unexpected error string: %q", err.Error())
	}
}

func TestAppError_Unwrap_ReturnsWrappedError(t *testing.T) {
	underlying := fmt.Errorf("original")
	wrapped := apperrors.ErrInternal.Wrap(underlying)

	if wrapped.Unwrap() != underlying { //nolint:errorlint // testing the Unwrap() mechanism directly
		t.Error("Unwrap() did not return original error")
	}
}

func TestAppError_WithMessage_ReturnsNewMessageKeepsCode(t *testing.T) {
	custom := apperrors.ErrValidation.WithMessage("email is invalid")

	if custom.Message != "email is invalid" {
		t.Errorf("expected custom message, got %q", custom.Message)
	}
	if custom.Code != apperrors.ErrValidation.Code {
		t.Errorf("expected same code %q, got %q", apperrors.ErrValidation.Code, custom.Code)
	}
}

func TestIsNotFound_ReturnsTrueForWrappedNotFound(t *testing.T) {
	err := apperrors.ErrNotFound.Wrap(fmt.Errorf("missing record"))
	if !apperrors.IsNotFound(err) {
		t.Error("expected IsNotFound to return true")
	}
}

func TestIsNotFound_ReturnsFalseForOtherErrors(t *testing.T) {
	if apperrors.IsNotFound(apperrors.ErrInternal) {
		t.Error("expected IsNotFound to return false for ErrInternal")
	}
}

func TestIsNotFound_ReturnsFalseForPlainError(t *testing.T) {
	if apperrors.IsNotFound(fmt.Errorf("plain error")) {
		t.Error("expected IsNotFound to return false for plain error")
	}
}

func TestIsUnauthorized_ReturnsTrueForUnauthorized(t *testing.T) {
	if !apperrors.IsUnauthorized(apperrors.ErrUnauthorized) {
		t.Error("expected IsUnauthorized to return true")
	}
}

func TestIsUnauthorized_ReturnsFalseForPlainError(t *testing.T) {
	if apperrors.IsUnauthorized(fmt.Errorf("plain error")) {
		t.Error("expected IsUnauthorized to return false for plain error")
	}
}

func TestAsAppError_ReturnsAppErrorWhenPresent(t *testing.T) {
	err := apperrors.ErrNotFound
	result := apperrors.AsAppError(err)

	if result.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %q", result.Code)
	}
}

func TestAsAppError_WrapsUnknownErrorAsInternal(t *testing.T) {
	err := fmt.Errorf("some unknown error")
	result := apperrors.AsAppError(err)

	if result.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", result.HTTPStatus)
	}
	if result.Code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR, got %q", result.Code)
	}
}

func TestSentinelErrors_HaveCorrectHTTPStatus(t *testing.T) {
	cases := []struct {
		err    *apperrors.AppError
		status int
	}{
		{apperrors.ErrNotFound, http.StatusNotFound},
		{apperrors.ErrUnauthorized, http.StatusUnauthorized},
		{apperrors.ErrForbidden, http.StatusForbidden},
		{apperrors.ErrValidation, http.StatusUnprocessableEntity},
		{apperrors.ErrConflict, http.StatusConflict},
		{apperrors.ErrInternal, http.StatusInternalServerError},
		{apperrors.ErrBadRequest, http.StatusBadRequest},
	}

	for _, tc := range cases {
		if tc.err.HTTPStatus != tc.status {
			t.Errorf("%s: expected status %d, got %d", tc.err.Code, tc.status, tc.err.HTTPStatus)
		}
	}
}
