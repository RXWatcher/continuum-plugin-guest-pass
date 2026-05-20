package store

import (
	"strings"
	"testing"
)

func TestGenerateTokenAndHashTokenAgree(t *testing.T) {
	token, hash, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" || hash == "" {
		t.Fatalf("token and hash must be populated")
	}
	if token == hash {
		t.Fatalf("token and hash should not match")
	}
	if got := HashToken(token); got != hash {
		t.Fatalf("HashToken(token) = %q, want %q", got, hash)
	}
}

func TestVerifyPINHandlesBcryptAndLegacy(t *testing.T) {
	hash, err := HashPIN("4242")
	if err != nil {
		t.Fatalf("HashPIN: %v", err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("bcrypt hash should start with $2, got %q", hash)
	}

	if got := VerifyPIN(hash, "4242"); got != PINVerifyOK {
		t.Fatalf("VerifyPIN(bcrypt match) = %v, want PINVerifyOK", got)
	}
	if got := VerifyPIN(hash, "9999"); got != PINVerifyFail {
		t.Fatalf("VerifyPIN(bcrypt wrong) = %v, want PINVerifyFail", got)
	}

	legacy := legacyPINHash("4242")
	if got := VerifyPIN(legacy, "4242"); got != PINVerifyOKRehash {
		t.Fatalf("VerifyPIN(legacy match) = %v, want PINVerifyOKRehash", got)
	}
	if got := VerifyPIN(legacy, "9999"); got != PINVerifyFail {
		t.Fatalf("VerifyPIN(legacy wrong) = %v, want PINVerifyFail", got)
	}

	// Empty stored hash should accept any PIN (PIN not enabled).
	if got := VerifyPIN("", ""); got != PINVerifyOK {
		t.Fatalf("VerifyPIN(empty) = %v, want PINVerifyOK", got)
	}
}

func TestHashPINEmptyInputProducesEmptyHash(t *testing.T) {
	got, err := HashPIN("")
	if err != nil {
		t.Fatalf("HashPIN(\"\"): %v", err)
	}
	if got != "" {
		t.Fatalf("HashPIN(\"\") = %q, want empty", got)
	}
}
