package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    uint32 = 1
	argonMemory  uint32 = 64 * 1024 // 64 MiB
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	saltLen             = 16
)

// HashPassword creates an argon2id hash of password+pepper.
// Returns "base64(salt):base64(hash)".
// The pepper is a server-side secret and is never stored.
func HashPassword(password, pepper string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}
	hash := argon2.IDKey([]byte(pepper+password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return base64.StdEncoding.EncodeToString(salt) + ":" + base64.StdEncoding.EncodeToString(hash), nil
}

// VerifyPassword returns true if password+pepper matches the stored hash.
// Uses constant-time comparison to prevent timing attacks.
func VerifyPassword(password, pepper, stored string) bool {
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expectedHash, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(pepper+password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return subtle.ConstantTimeCompare(actual, expectedHash) == 1
}
