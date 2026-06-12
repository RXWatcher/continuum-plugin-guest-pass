package server

import (
	"sync"
	"time"
)

// redemptionLimiter is a general per-(pass, client) token-bucket rate
// limiter for the public /open and /play endpoints. It throttles redemption
// abuse (rapid retry loops, scripted hammering) independently of, and in
// addition to, the DB-backed bad-PIN lockout.
//
// Keying is per (passID, client) where client is the resolved client IP when
// available, falling back to the device id so a request with no host-stamped
// IP is still throttled per device. This bounds how fast any single guest can
// pound a given pass without affecting other guests or other passes.
//
// The implementation is in-memory and lock-protected. It is intentionally not
// DB-backed: the limiter runs on the hot path of every public request and must
// not add a query per call. A single plugin process serves the public route,
// so a process-local bucket is the right scope; the DB-backed bad-PIN lockout
// already provides cross-restart durability for the security-sensitive case.
type redemptionLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	// rate is tokens added per second; burst is the bucket capacity.
	rate     float64
	burst    float64
	now      func() time.Time
	lastGC   time.Time
	gcEvery  time.Duration
	idleEvic time.Duration
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

// newRedemptionLimiter builds a limiter allowing burst requests immediately
// and refilling at rate tokens/sec per key.
func newRedemptionLimiter(rate, burst float64) *redemptionLimiter {
	if rate <= 0 {
		rate = 1
	}
	if burst < 1 {
		burst = 1
	}
	return &redemptionLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     rate,
		burst:    burst,
		now:      time.Now,
		gcEvery:  5 * time.Minute,
		idleEvic: 10 * time.Minute,
	}
}

// Allow reports whether a request for key may proceed, consuming one token
// when it returns true.
func (l *redemptionLimiter) Allow(key string) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	l.gcLocked(now)

	b := l.buckets[key]
	if b == nil {
		// New key starts full, minus the token this request consumes.
		l.buckets[key] = &tokenBucket{tokens: l.burst - 1, last: now}
		return true
	}

	// Refill based on elapsed time, capped at burst.
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * l.rate
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.last = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// gcLocked periodically evicts idle buckets so the map doesn't grow without
// bound under a churn of distinct keys. Caller must hold l.mu.
func (l *redemptionLimiter) gcLocked(now time.Time) {
	if now.Sub(l.lastGC) < l.gcEvery {
		return
	}
	l.lastGC = now
	for k, b := range l.buckets {
		if now.Sub(b.last) > l.idleEvic {
			delete(l.buckets, k)
		}
	}
}
