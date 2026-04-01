package auth_test

import (
	"strings"
	"testing"

	"github.com/rian/infinite_brain/internal/auth"
)

func TestHashPassword_ProducesVerifiableHash(t *testing.T) {
	hash, err := auth.HashPassword("my-password", "pepper123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if !strings.Contains(hash, ":") {
		t.Errorf("expected salt:hash format, got %q", hash)
	}
	if !auth.VerifyPassword("my-password", "pepper123", hash) {
		t.Error("VerifyPassword returned false for correct password")
	}
}

func TestHashPassword_DifferentPepperFails(t *testing.T) {
	hash, err := auth.HashPassword("my-password", "pepper123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if auth.VerifyPassword("my-password", "wrong-pepper", hash) {
		t.Error("VerifyPassword returned true for wrong pepper")
	}
}

func TestHashPassword_DifferentPasswordFails(t *testing.T) {
	hash, err := auth.HashPassword("my-password", "pepper123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if auth.VerifyPassword("wrong-password", "pepper123", hash) {
		t.Error("VerifyPassword returned true for wrong password")
	}
}

func TestHashPassword_TwoCallsProduceDifferentHashes(t *testing.T) {
	h1, _ := auth.HashPassword("password", "pepper")
	h2, _ := auth.HashPassword("password", "pepper")
	if h1 == h2 {
		t.Error("expected different hashes from two calls (random salt)")
	}
}
