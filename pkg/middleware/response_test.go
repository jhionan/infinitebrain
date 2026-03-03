package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/rian/infinite_brain/pkg/errors"
	"github.com/rian/infinite_brain/pkg/middleware"
)

func TestJSON_WritesCorrectStatusAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	middleware.JSON(w, http.StatusOK, map[string]string{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp middleware.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got %v", resp.Error)
	}
}

func TestJSONError_MapsAppErrorToCorrectStatus(t *testing.T) {
	w := httptest.NewRecorder()
	middleware.JSONError(w, apperrors.ErrNotFound)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var resp middleware.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error in response, got nil")
	}
	if resp.Error.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND code, got %q", resp.Error.Code)
	}
}

func TestJSONError_UnknownErrorReturns500(t *testing.T) {
	w := httptest.NewRecorder()
	middleware.JSONError(w, apperrors.ErrInternal)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestJSONWithMeta_IncludesMetaInResponse(t *testing.T) {
	w := httptest.NewRecorder()
	meta := &middleware.Meta{Total: 42, NextCursor: "cursor-abc"}
	middleware.JSONWithMeta(w, http.StatusOK, []string{"a", "b"}, meta)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Meta *middleware.Meta `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Meta == nil {
		t.Fatal("expected meta, got nil")
	}
	if resp.Meta.Total != 42 {
		t.Errorf("expected total 42, got %d", resp.Meta.Total)
	}
	if resp.Meta.NextCursor != "cursor-abc" {
		t.Errorf("expected next_cursor 'cursor-abc', got %q", resp.Meta.NextCursor)
	}
}
