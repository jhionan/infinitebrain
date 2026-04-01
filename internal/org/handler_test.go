package org_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/internal/auth"
	"github.com/rian/infinite_brain/internal/org"
)

var testSigner = auth.NewSigner("test-secret-that-is-32chars-long!!", 15*time.Minute)

// injectClaims adds auth claims to the request context by going through
// the real auth middleware with a freshly signed token.
func injectClaims(r *http.Request, userID uuid.UUID) *http.Request {
	user := &auth.User{
		ID:    userID,
		OrgID: uuid.New(),
		Email: "test@example.com",
		Role:  "owner",
	}
	token, err := testSigner.Sign(user)
	if err != nil {
		panic("injectClaims: sign failed: " + err.Error())
	}
	r.Header.Set("Authorization", "Bearer "+token)
	// Manually inject via real middleware so the context key is set correctly.
	var injected *http.Request
	auth.Auth(testSigner)(http.HandlerFunc(func(_ http.ResponseWriter, rr *http.Request) {
		injected = rr
	})).ServeHTTP(httptest.NewRecorder(), r)
	return injected
}

func newOrgHandler() (*org.Handler, *mockRepo) {
	r := newMockRepo()
	svc := org.NewService(r)
	return org.NewHandler(svc, zerolog.Nop()), r
}

func TestOrgHandler_GetOrg_Returns200(t *testing.T) {
	h, repo := newOrgHandler()
	o := &org.Org{ID: uuid.New(), Name: "Test", Slug: "test", Plan: "personal", CreatedAt: time.Now()}
	repo.seedOrg(o)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/orgs/{slug}", h.GetOrg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data object in response")
	}
	if data["slug"] != "test" {
		t.Errorf("expected slug 'test', got %v", data["slug"])
	}
}

func TestOrgHandler_GetOrg_Returns404ForMissing(t *testing.T) {
	h, _ := newOrgHandler()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/orgs/{slug}", h.GetOrg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/no-such", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgHandler_AddMember_Returns204(t *testing.T) {
	h, repo := newOrgHandler()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "add-mem", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/orgs/{slug}/members", h.AddMember)

	body, _ := json.Marshal(map[string]string{"user_id": uuid.New().String(), "role": "editor"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/add-mem/members", bytes.NewReader(body))
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrgHandler_ListMembers_Returns401WithNoAuth(t *testing.T) {
	h, repo := newOrgHandler()
	o := &org.Org{ID: uuid.New(), Slug: "list-noauth", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/orgs/{slug}/members", h.ListMembers)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/list-noauth/members", nil)
	// No auth claims injected
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestOrgHandler_AddMember_Returns401WithNoAuth(t *testing.T) {
	h, repo := newOrgHandler()
	o := &org.Org{ID: uuid.New(), Slug: "noauth", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/orgs/{slug}/members", h.AddMember)

	body, _ := json.Marshal(map[string]string{"user_id": uuid.New().String(), "role": "editor"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/noauth/members", bytes.NewReader(body))
	// No auth claims injected
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
