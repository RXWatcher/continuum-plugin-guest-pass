package store

import (
	"context"
	"strings"
)

// RegisterDevice records that deviceID was seen for passID and enforces
// MaxDevices atomically by locking the parent pass row before counting and
// inserting. Empty deviceID or non-positive maxDevices is a no-op.
//
// Concurrent registration of the *same* deviceID is a no-op via the
// (pass_id, device_id) unique index; concurrent registration of *different*
// device IDs is serialised through SELECT ... FOR UPDATE on the parent row.
func (s *Store) RegisterDevice(ctx context.Context, passID int, deviceID, ip, ua string, maxDevices int) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || maxDevices <= 0 {
		return nil
	}
	ua = trimLen(ua, 512)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT 1 FROM guest_passes WHERE id = $1 FOR UPDATE`, passID); err != nil {
		return err
	}

	// Fast path: existing device, just bump last_seen.
	tag, err := tx.Exec(ctx, `
UPDATE guest_pass_devices
SET last_seen_at = NOW(),
    first_ip     = COALESCE(NULLIF(first_ip, ''), $3),
    user_agent   = $4
WHERE pass_id = $1 AND device_id = $2`, passID, deviceID, ip, ua)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return tx.Commit(ctx)
	}

	// New device: enforce the cap under the same lock.
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM guest_pass_devices WHERE pass_id = $1`, passID).Scan(&count); err != nil {
		return err
	}
	if count >= maxDevices {
		return ErrDeviceLimit
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO guest_pass_devices (pass_id, device_id, first_ip, user_agent) VALUES ($1,$2,$3,$4)`,
		passID, deviceID, ip, ua); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
