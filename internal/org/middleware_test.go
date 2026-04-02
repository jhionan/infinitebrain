package org_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/org"
)

func TestOrgResolver_SubdomainPresent_InjectsOrgIntoContext(t *testing.T) {
	repo := newMockRepo()
	o := &org.Org{ID: uuid.New(), Slug: "acme", Plan: "teams", CreatedAt: time.Now()}
	repo.seedOrg(o)

	var gotOrg *org.Org
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = org.OrgFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := org.OrgResolver(repo)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "acme.infinitebrain.io"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if gotOrg == nil || gotOrg.Slug != "acme" {
		t.Errorf("expected org acme in context, got %v", gotOrg)
	}
}

func TestOrgResolver_NoSubdomain_PassesThrough(t *testing.T) {
	repo := newMockRepo()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := org.OrgResolver(repo)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "infinitebrain.io"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("next handler not called for non-subdomain request")
	}
}

func TestOrgResolver_UnknownSubdomain_Returns404(t *testing.T) {
	repo := newMockRepo()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := org.OrgResolver(repo)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "unknown-slug.infinitebrain.io"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgResolver_WWWSubdomain_PassesThrough(t *testing.T) {
	repo := newMockRepo()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := org.OrgResolver(repo)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "www.infinitebrain.io"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("next handler not called for www subdomain")
	}
}
