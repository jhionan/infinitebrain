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

func TestOrgHandler_UpdateOrg_Returns200(t *testing.T) {
	h, repo := newOrgHandler()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "update-slug", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "admin", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/orgs/{slug}", h.UpdateOrg)

	body, _ := json.Marshal(map[string]any{"name": "Updated Name", "settings": map[string]any{}})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orgs/update-slug", bytes.NewReader(body))
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrgHandler_UpdateOrg_Returns401WithNoAuth(t *testing.T) {
	h, repo := newOrgHandler()
	o := &org.Org{ID: uuid.New(), Slug: "update-noauth", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/orgs/{slug}", h.UpdateOrg)

	body, _ := json.Marshal(map[string]any{"name": "x", "settings": map[string]any{}})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orgs/update-noauth", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestOrgHandler_UpdateOrg_Returns404ForMissingOrg(t *testing.T) {
	h, _ := newOrgHandler()
	ownerID := uuid.New()

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/orgs/{slug}", h.UpdateOrg)

	body, _ := json.Marshal(map[string]any{"name": "x", "settings": map[string]any{}})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orgs/no-such-slug", bytes.NewReader(body))
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgHandler_ListMembers_Returns200(t *testing.T) {
	h, repo := newOrgHandler()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "list-ok", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/orgs/{slug}/members", h.ListMembers)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/list-ok/members", nil)
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrgHandler_UpdateMemberRole_Scenarios(t *testing.T) {
	tests := []struct {
		name       string
		setupSlug  string
		callerRole string
		wantCode   int
		injectAuth bool
	}{
		{"owner updates editor role", "updaterole-ok", "owner", http.StatusNoContent, true},
		{"viewer is forbidden", "updaterole-forbidden-h", "viewer", http.StatusForbidden, true},
		{"no auth returns 401", "updaterole-noauth", "", http.StatusUnauthorized, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, repo := newOrgHandler()
			callerID := uuid.New()
			targetID := uuid.New()
			o := &org.Org{ID: uuid.New(), Slug: tt.setupSlug, Plan: "teams", CreatedAt: time.Now()}
			repo.seedOrg(o)
			if tt.callerRole != "" {
				_ = repo.AddMember(context.Background(), o.ID, callerID, tt.callerRole, nil)
			}
			_ = repo.AddMember(context.Background(), o.ID, targetID, "editor", nil)

			mux := http.NewServeMux()
			mux.HandleFunc("PUT /api/v1/orgs/{slug}/members/{userID}", h.UpdateMemberRole)

			body, _ := json.Marshal(map[string]string{"role": "viewer"})
			req := httptest.NewRequest(http.MethodPut,
				"/api/v1/orgs/"+tt.setupSlug+"/members/"+targetID.String(),
				bytes.NewReader(body))
			if tt.injectAuth {
				req = injectClaims(req, callerID)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected %d, got %d: %s", tt.wantCode, w.Code, w.Body.String())
			}
		})
	}
}

func TestOrgHandler_RemoveMember_Scenarios(t *testing.T) {
	tests := []struct {
		name       string
		setupSlug  string
		callerRole string
		wantCode   int
		injectAuth bool
	}{
		{"owner removes editor", "remove-ok", "owner", http.StatusNoContent, true},
		{"viewer is forbidden", "remove-forbidden-h", "viewer", http.StatusForbidden, true},
		{"no auth returns 401", "remove-noauth", "", http.StatusUnauthorized, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, repo := newOrgHandler()
			callerID := uuid.New()
			targetID := uuid.New()
			o := &org.Org{ID: uuid.New(), Slug: tt.setupSlug, Plan: "teams", CreatedAt: time.Now()}
			repo.seedOrg(o)
			if tt.callerRole != "" {
				_ = repo.AddMember(context.Background(), o.ID, callerID, tt.callerRole, nil)
			}
			_ = repo.AddMember(context.Background(), o.ID, targetID, "editor", nil)

			mux := http.NewServeMux()
			mux.HandleFunc("DELETE /api/v1/orgs/{slug}/members/{userID}", h.RemoveMember)

			req := httptest.NewRequest(http.MethodDelete,
				"/api/v1/orgs/"+tt.setupSlug+"/members/"+targetID.String(), nil)
			if tt.injectAuth {
				req = injectClaims(req, callerID)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected %d, got %d: %s", tt.wantCode, w.Code, w.Body.String())
			}
		})
	}
}

func TestOrgHandler_AddMember_Returns400OnBadJSON(t *testing.T) {
	h, repo := newOrgHandler()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "addbad-json", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/orgs/{slug}/members", h.AddMember)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/addbad-json/members",
		bytes.NewReader([]byte(`{bad json`)))
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrgHandler_AddMember_Returns400OnBadUserID(t *testing.T) {
	h, repo := newOrgHandler()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "addbad-userid", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/orgs/{slug}/members", h.AddMember)

	body, _ := json.Marshal(map[string]string{"user_id": "not-a-uuid", "role": "editor"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/addbad-userid/members",
		bytes.NewReader(body))
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrgHandler_AddMember_Returns404ForMissingOrg(t *testing.T) {
	h, _ := newOrgHandler()
	ownerID := uuid.New()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/orgs/{slug}/members", h.AddMember)

	body, _ := json.Marshal(map[string]string{"user_id": uuid.New().String(), "role": "editor"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/no-such/members",
		bytes.NewReader(body))
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgHandler_UpdateMemberRole_Returns404ForMissingOrg(t *testing.T) {
	h, _ := newOrgHandler()
	ownerID := uuid.New()

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/orgs/{slug}/members/{userID}", h.UpdateMemberRole)

	body, _ := json.Marshal(map[string]string{"role": "viewer"})
	req := httptest.NewRequest(http.MethodPut,
		"/api/v1/orgs/no-such/members/"+uuid.New().String(),
		bytes.NewReader(body))
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgHandler_UpdateMemberRole_Returns400OnBadJSON(t *testing.T) {
	h, repo := newOrgHandler()
	ownerID := uuid.New()
	targetID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "updaterole-badjson", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)
	_ = repo.AddMember(context.Background(), o.ID, targetID, "editor", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/orgs/{slug}/members/{userID}", h.UpdateMemberRole)

	req := httptest.NewRequest(http.MethodPut,
		"/api/v1/orgs/updaterole-badjson/members/"+targetID.String(),
		bytes.NewReader([]byte(`{bad json`)))
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrgHandler_RemoveMember_Returns404ForMissingOrg(t *testing.T) {
	h, _ := newOrgHandler()
	ownerID := uuid.New()

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/orgs/{slug}/members/{userID}", h.RemoveMember)

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/orgs/no-such/members/"+uuid.New().String(), nil)
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgHandler_UpdateMemberRole_Returns422OnInvalidTargetID(t *testing.T) {
	h, repo := newOrgHandler()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "updaterole-badtarget", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/orgs/{slug}/members/{userID}", h.UpdateMemberRole)

	body, _ := json.Marshal(map[string]string{"role": "viewer"})
	req := httptest.NewRequest(http.MethodPut,
		"/api/v1/orgs/updaterole-badtarget/members/not-a-uuid",
		bytes.NewReader(body))
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrgHandler_RemoveMember_Returns422OnInvalidTargetID(t *testing.T) {
	h, repo := newOrgHandler()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "remove-badtarget", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/orgs/{slug}/members/{userID}", h.RemoveMember)

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/orgs/remove-badtarget/members/not-a-uuid", nil)
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrgHandler_ListMembers_Returns404ForMissingOrg(t *testing.T) {
	h, _ := newOrgHandler()
	ownerID := uuid.New()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/orgs/{slug}/members", h.ListMembers)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/no-such/members", nil)
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgHandler_UpdateOrg_Returns400OnBadJSON(t *testing.T) {
	h, repo := newOrgHandler()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "update-badjson", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "admin", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/orgs/{slug}", h.UpdateOrg)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/orgs/update-badjson",
		bytes.NewReader([]byte(`{bad json`)))
	req = injectClaims(req, ownerID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
