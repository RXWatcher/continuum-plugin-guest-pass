package server

import (
	"testing"
	"time"
)

func TestRedemptionLimiterBurstThenThrottle(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	now := base
	l := newRedemptionLimiter(1.0/3.0, 5)
	l.now = func() time.Time { return now }

	// The first burst (= capacity) of requests on a fresh key all pass.
	for i := 0; i < 5; i++ {
		if !l.Allow("pass|ip") {
			t.Fatalf("request %d within burst should be allowed", i)
		}
	}
	// The next request, with no time elapsed, is throttled.
	if l.Allow("pass|ip") {
		t.Fatal("request beyond burst with no refill should be throttled")
	}
}

func TestRedemptionLimiterRefillsOverTime(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	now := base
	l := newRedemptionLimiter(1.0/3.0, 2) // refill 1 token per 3s
	l.now = func() time.Time { return now }

	// Drain the burst.
	if !l.Allow("k") || !l.Allow("k") {
		t.Fatal("initial burst should be allowed")
	}
	if l.Allow("k") {
		t.Fatal("should be throttled after draining burst")
	}

	// Advance 3s: exactly one token should be available again.
	now = now.Add(3 * time.Second)
	if !l.Allow("k") {
		t.Fatal("one token should have refilled after 3s")
	}
	if l.Allow("k") {
		t.Fatal("only one token should have refilled")
	}
}

func TestRedemptionLimiterKeysAreIndependent(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	now := base
	l := newRedemptionLimiter(1.0/3.0, 1)
	l.now = func() time.Time { return now }

	if !l.Allow("a") {
		t.Fatal("key a first request should pass")
	}
	// a is now drained, but b is a distinct key and should still pass.
	if !l.Allow("b") {
		t.Fatal("key b should be independent of key a")
	}
	if l.Allow("a") {
		t.Fatal("key a should remain throttled")
	}
}

func TestRedemptionLimiterEvictsIdleBuckets(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	now := base
	l := newRedemptionLimiter(1.0/3.0, 1)
	l.now = func() time.Time { return now }

	l.Allow("old")
	if len(l.buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(l.buckets))
	}

	// Advance past the GC interval and the idle-eviction window, then touch a
	// new key to trigger gcLocked, which should drop the idle "old" bucket.
	now = now.Add(l.gcEvery + l.idleEvic + time.Second)
	l.Allow("new")
	if _, ok := l.buckets["old"]; ok {
		t.Fatal("idle bucket should have been evicted")
	}
	if _, ok := l.buckets["new"]; !ok {
		t.Fatal("new bucket should be present")
	}
}
