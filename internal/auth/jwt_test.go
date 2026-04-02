package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/auth"
)

func TestSigner_SignAndVerify_ValidToken(t *testing.T) {
	signer := auth.NewSigner("super-secret-key-32-chars-minimum!!", 15*time.Minute)
	user := &auth.User{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Email: "test@example.com",
		Role:  "owner",
	}

	token, err := signer.Sign(user)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if token == "" {
		t.Fatal("Sign returned empty token")
	}

	claims, err := signer.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", claims.UserID, user.ID)
	}
	if claims.Email != user.Email {
		t.Errorf("Email = %q, want %q", claims.Email, user.Email)
	}
	if claims.Role != user.Role {
		t.Errorf("Role = %q, want %q", claims.Role, user.Role)
	}
}

func TestSigner_Verify_RejectsExpiredToken(t *testing.T) {
	signer := auth.NewSigner("super-secret-key-32-chars-minimum!!", -1*time.Second)
	user := &auth.User{ID: uuid.New(), OrgID: uuid.New(), Email: "e@x.com", Role: "member"}

	token, err := signer.Sign(user)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	_, err = signer.Verify(token)
	if err == nil {
		t.Error("Verify should have rejected expired token")
	}
}

func TestSigner_Verify_RejectsTamperedToken(t *testing.T) {
	signer := auth.NewSigner("super-secret-key-32-chars-minimum!!", 15*time.Minute)
	user := &auth.User{ID: uuid.New(), OrgID: uuid.New(), Email: "e@x.com", Role: "member"}

	token, _ := signer.Sign(user)
	_, err := signer.Verify(token + "tampered")
	if err == nil {
		t.Error("Verify should have rejected tampered token")
	}
}

func TestSigner_Sign_EachCallHasUniqueJTI(t *testing.T) {
	signer := auth.NewSigner("super-secret-key-32-chars-minimum!!", 15*time.Minute)
	user := &auth.User{ID: uuid.New(), OrgID: uuid.New(), Email: "e@x.com", Role: "member"}

	t1, _ := signer.Sign(user)
	t2, _ := signer.Sign(user)
	if t1 == t2 {
		t.Error("two Sign calls should produce different tokens (unique JTI)")
	}
}
