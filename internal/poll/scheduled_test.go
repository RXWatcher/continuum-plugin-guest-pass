package poll

import (
	"context"
	"testing"
)

// Before the store is wired the scheduled task must be a no-op success so the
// SDK's first scheduled invocation during early lifecycle doesn't error.
func TestRunNoopBeforeStoreWired(t *testing.T) {
	s := New()
	resp, err := s.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run before Set should not error, got %v", err)
	}
	if resp == nil {
		t.Fatal("Run should return a non-nil response")
	}
}

// The grant grace constant must be non-negative so PruneExpiredGrants is never
// asked to delete future-expiring (still-active) grants.
func TestExpiredGrantGraceIsNonNegative(t *testing.T) {
	if expiredGrantGraceHours < 0 {
		t.Fatalf("expiredGrantGraceHours = %d, must be >= 0", expiredGrantGraceHours)
	}
}
