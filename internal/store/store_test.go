package store

import (
	"testing"
	"time"
)

func TestGenerateTokenReturnsHashOnlyComparable(t *testing.T) {
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

func TestPassStatusUsesEffectiveExpiryAndLimits(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	opened := now.Add(-2 * time.Hour)
	p := Pass{
		ExpiresAt:                now.Add(24 * time.Hour),
		FirstOpenedAt:            &opened,
		ValidHoursAfterFirstOpen: 1,
	}
	if got := p.Status(now); got != "expired" {
		t.Fatalf("Status with first-open expiry = %q, want expired", got)
	}

	p = Pass{ExpiresAt: now.Add(time.Hour), MaxOpens: 1, OpenCount: 1}
	if got := p.Status(now); got != "active" {
		t.Fatalf("Status after allowed open = %q, want active", got)
	}
	p.OpenCount = 2
	if got := p.Status(now); got != "open_limit_reached" {
		t.Fatalf("Status beyond open limit = %q, want open_limit_reached", got)
	}

	p = Pass{ExpiresAt: now.Add(time.Hour), MaxPlays: 1, PlayCount: 1}
	if got := p.Status(now); got != "play_limit_reached" {
		t.Fatalf("Status at play limit = %q, want play_limit_reached", got)
	}
}

func TestPINAndIPPolicyHelpers(t *testing.T) {
	p := Pass{RequirePIN: true, PINHash: HashPIN("1234")}
	if !p.PINMatches("1234") {
		t.Fatalf("PINMatches should accept the configured PIN")
	}
	if p.PINMatches("9999") {
		t.Fatalf("PINMatches should reject the wrong PIN")
	}

	p = Pass{IPAllowlist: []string{"192.0.2.0/24", "203.0.113.9"}}
	if !p.AllowsIP("192.0.2.55") {
		t.Fatalf("AllowsIP should accept CIDR matches")
	}
	if !p.AllowsIP("203.0.113.9") {
		t.Fatalf("AllowsIP should accept exact IP matches")
	}
	if p.AllowsIP("198.51.100.2") {
		t.Fatalf("AllowsIP should reject non-matches")
	}
}
