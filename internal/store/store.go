// Package store wraps the plugin's Postgres access layer. Each file in the
// package owns one bounded concern (passes, events, devices, grants, config,
// catalog) so handlers don't see a single 800-line surface.
//
// All public methods take a context.Context as the first arg and return
// well-typed sentinel errors (ErrNotFound, ErrOpenLimit, etc.) instead of
// leaking driver-level errors to callers.
package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors returned by store methods. Handlers map these to status
// codes; never compare on string content.
var (
	ErrNotFound         = errors.New("guest pass not found")
	ErrOpenLimit        = errors.New("open limit reached")
	ErrPlayLimit        = errors.New("play limit reached")
	ErrDeviceLimit      = errors.New("device limit reached")
	ErrConcurrencyLimit = errors.New("concurrent stream limit reached")
)

// Store holds the connection pool. Construct with New.
type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Pool exposes the underlying pool for code that needs a transaction in
// the same connection. Most callers should use the typed methods instead.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

func trimLen(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func defaultPositive(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
