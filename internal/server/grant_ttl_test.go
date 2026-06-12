package server

import (
	"testing"
	"time"

	"github.com/RXWatcher/silo-plugin-guest-pass/internal/store"
)

func TestCappedGrantExpiry(t *testing.T) {
	now := time.Now()

	// Far-future pass expiry is capped to now+grantTTLCap so a minted grant
	// can't outlive a revocation by more than the cap.
	far := &store.Pass{ExpiresAt: now.Add(72 * time.Hour)}
	got := cappedGrantExpiry(far, now)
	if want := now.Add(grantTTLCap); !got.Equal(want) {
		t.Fatalf("far expiry: got %s, want capped %s", got, want)
	}

	// A pass expiring sooner than the cap keeps its own (earlier) expiry.
	soon := &store.Pass{ExpiresAt: now.Add(time.Minute)}
	got = cappedGrantExpiry(soon, now)
	if want := now.Add(time.Minute); !got.Equal(want) {
		t.Fatalf("near expiry: got %s, want %s", got, want)
	}
}
