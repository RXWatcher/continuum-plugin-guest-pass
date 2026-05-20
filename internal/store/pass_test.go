package store

import (
	"testing"
	"time"
)

func TestPassStatusUsesEffectiveExpiry(t *testing.T) {
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
}

func TestPassStatusLimitsAreInclusiveForOpensAndPlays(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)

	// At-cap should report limit reached for both counters (consistent semantics).
	p := Pass{ExpiresAt: now.Add(time.Hour), MaxOpens: 1, OpenCount: 1}
	if got := p.Status(now); got != "open_limit_reached" {
		t.Fatalf("OpenCount=MaxOpens should be open_limit_reached, got %q", got)
	}
	p = Pass{ExpiresAt: now.Add(time.Hour), MaxPlays: 1, PlayCount: 1}
	if got := p.Status(now); got != "play_limit_reached" {
		t.Fatalf("PlayCount=MaxPlays should be play_limit_reached, got %q", got)
	}

	// Below cap is active.
	p = Pass{ExpiresAt: now.Add(time.Hour), MaxOpens: 2, OpenCount: 1}
	if got := p.Status(now); got != "active" {
		t.Fatalf("OpenCount<MaxOpens should be active, got %q", got)
	}
}

func TestPassPINMatch(t *testing.T) {
	hash, err := HashPIN("1234")
	if err != nil {
		t.Fatalf("HashPIN: %v", err)
	}
	p := Pass{RequirePIN: true, PINHash: hash}

	matched, rehash := p.PINMatch("1234")
	if !matched || rehash {
		t.Fatalf("PINMatch(correct) = (%v,%v), want (true,false)", matched, rehash)
	}
	matched, _ = p.PINMatch("9999")
	if matched {
		t.Fatalf("PINMatch should reject wrong PIN")
	}

	// Pass without RequirePIN always matches and never asks for rehash.
	p.RequirePIN = false
	matched, rehash = p.PINMatch("")
	if !matched || rehash {
		t.Fatalf("PINMatch on non-PIN pass = (%v,%v), want (true,false)", matched, rehash)
	}
}

func TestPassPINMatchLegacyHashRequestsRehash(t *testing.T) {
	p := Pass{RequirePIN: true, PINHash: legacyPINHash("4242")}
	matched, rehash := p.PINMatch("4242")
	if !matched || !rehash {
		t.Fatalf("legacy match should be (true,true), got (%v,%v)", matched, rehash)
	}
}

func TestPassAllowsIP(t *testing.T) {
	p := Pass{IPAllowlist: []string{"192.0.2.0/24", "203.0.113.9"}}
	if !p.AllowsIP("192.0.2.55") {
		t.Fatalf("AllowsIP should accept CIDR matches")
	}
	if !p.AllowsIP("203.0.113.9") {
		t.Fatalf("AllowsIP should accept exact IP matches")
	}
	if p.AllowsIP("198.51.100.2") {
		t.Fatalf("AllowsIP should reject non-matches")
	}
	// Empty allowlist allows everything.
	if !(Pass{}).AllowsIP("198.51.100.2") {
		t.Fatalf("empty allowlist should allow all")
	}
}
