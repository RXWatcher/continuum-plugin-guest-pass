package store

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// GenerateToken returns a fresh URL-safe token and its sha256 hash. Only the
// hash is stored; the plaintext is shown once to the operator.
func GenerateToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token), nil
}

// HashToken returns the sha256 hex of a guest-pass token. Tokens are
// 32 bytes of entropy, so sha256 without a work factor is sufficient —
// pre-image search at that entropy is infeasible.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// HashPIN hashes a PIN with bcrypt. PINs are low-entropy (often 4–6 digits)
// so a work factor is required; the legacy verify path still accepts pre-
// rewrite sha256 hashes and the caller is expected to rehash on match —
// see VerifyPIN.
func HashPIN(pin string) (string, error) {
	if pin == "" {
		return "", nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash pin: %w", err)
	}
	return string(hash), nil
}

// PINVerifyResult signals whether a PIN matched, and if so whether the
// stored hash should be upgraded (legacy sha256 → bcrypt).
type PINVerifyResult int

const (
	// PINVerifyFail means the supplied PIN did not match the stored hash.
	PINVerifyFail PINVerifyResult = iota
	// PINVerifyOK means the PIN matched a current-format (bcrypt) hash.
	PINVerifyOK
	// PINVerifyOKRehash means the PIN matched a legacy sha256 hash. The
	// caller should rehash via Store.RehashPIN to upgrade storage.
	PINVerifyOKRehash
)

// VerifyPIN compares a user-supplied PIN against a stored hash. Empty
// stored hash always matches (PIN was never set on this pass).
func VerifyPIN(stored, pin string) PINVerifyResult {
	if stored == "" {
		return PINVerifyOK
	}
	if strings.HasPrefix(stored, "$2") {
		if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(pin)); err == nil {
			return PINVerifyOK
		}
		return PINVerifyFail
	}
	legacy := legacyPINHash(pin)
	if subtle.ConstantTimeCompare([]byte(stored), []byte(legacy)) == 1 {
		return PINVerifyOKRehash
	}
	return PINVerifyFail
}

func legacyPINHash(pin string) string {
	sum := sha256.Sum256([]byte("pin:" + pin))
	return hex.EncodeToString(sum[:])
}
