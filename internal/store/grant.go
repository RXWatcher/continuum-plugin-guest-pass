package store

import (
	"context"
	"time"
)

// ReserveGrantSlot atomically enforces MaxConcurrentStreams and inserts a
// grant row, returning the new row's ID. Returns ErrConcurrencyLimit if the
// pass is already at its cap.
//
// Locks the parent pass row, so concurrent calls for the same pass are
// serialised — the count check and insert happen as one unit.
// maxConcurrent <= 0 means unlimited.
//
// If a downstream step fails (e.g. minting the host stream), call
// ReleaseGrantSlot with the returned ID so the slot doesn't count against
// the concurrency cap until expires_at.
func (s *Store) ReserveGrantSlot(ctx context.Context, passID int, deviceID string, expiresAt time.Time, maxConcurrent int) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT 1 FROM guest_passes WHERE id = $1 FOR UPDATE`, passID); err != nil {
		return 0, err
	}

	if maxConcurrent > 0 {
		var n int
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM guest_pass_grants WHERE pass_id = $1 AND expires_at > NOW()`, passID,
		).Scan(&n); err != nil {
			return 0, err
		}
		if n >= maxConcurrent {
			return 0, ErrConcurrencyLimit
		}
	}

	var id int
	if err := tx.QueryRow(ctx,
		`INSERT INTO guest_pass_grants (pass_id, device_id, expires_at) VALUES ($1,$2,$3) RETURNING id`,
		passID, trimLen(deviceID, 256), expiresAt).Scan(&id); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

// ReleaseGrantSlot deletes a previously reserved grant row. Used when a
// downstream step (e.g. host stream mint) fails after the slot was reserved.
func (s *Store) ReleaseGrantSlot(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM guest_pass_grants WHERE id = $1`, id)
	return err
}

// ActiveGrantCount is exposed for diagnostics; the playback flow uses
// ReserveGrantSlot which checks and inserts atomically.
func (s *Store) ActiveGrantCount(ctx context.Context, passID int) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM guest_pass_grants WHERE pass_id = $1 AND expires_at > NOW()`, passID).Scan(&count)
	return count, err
}
