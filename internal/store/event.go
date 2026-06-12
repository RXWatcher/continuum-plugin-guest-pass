package store

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"time"
)

// Event is one audit log row.
type Event struct {
	ID        int            `json:"id"`
	PassID    int            `json:"pass_id"`
	Type      string         `json:"type"`
	IP        string         `json:"ip,omitempty"`
	UserAgent string         `json:"user_agent,omitempty"`
	Attrs     map[string]any `json:"attrs,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// RecordEvent inserts a single audit row. Best-effort: the caller usually
// discards the error to avoid blocking the user flow on logging.
func (s *Store) RecordEvent(ctx context.Context, passID int, typ, ip, ua string, attrs map[string]any) error {
	if attrs == nil {
		attrs = map[string]any{}
	}
	if parsed := net.ParseIP(ip); parsed == nil && len(ip) > 256 {
		ip = ip[:256]
	}
	ua = trimLen(ua, 512)
	raw, err := json.Marshal(attrs)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO guest_pass_events (pass_id, event_type, ip, user_agent, attrs) VALUES ($1,$2,$3,$4,$5)`,
		passID, typ, ip, ua, raw)
	return err
}

// CountRecentBadPINs returns how many "*_rejected_bad_pin" audit rows were
// recorded for a pass within the trailing window. When ip is non-empty the
// count is additionally scoped to that client IP, so a single abusive IP
// can be locked out without affecting other legitimate guests of the same
// pass. Used by the PIN brute-force backoff in the public handlers.
func (s *Store) CountRecentBadPINs(ctx context.Context, passID int, ip string, window time.Duration) (int, error) {
	seconds := int(window.Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	var count int
	var err error
	if strings.TrimSpace(ip) != "" {
		err = s.pool.QueryRow(ctx, `
SELECT COUNT(*) FROM guest_pass_events
 WHERE pass_id = $1
   AND event_type LIKE '%_rejected_bad_pin'
   AND ip = $3
   AND created_at > NOW() - ($2::text || ' seconds')::interval`, passID, seconds, ip).Scan(&count)
	} else {
		err = s.pool.QueryRow(ctx, `
SELECT COUNT(*) FROM guest_pass_events
 WHERE pass_id = $1
   AND event_type LIKE '%_rejected_bad_pin'
   AND created_at > NOW() - ($2::text || ' seconds')::interval`, passID, seconds).Scan(&count)
	}
	return count, err
}

// ListEvents returns the 200 most recent audit rows for a pass.
func (s *Store) ListEvents(ctx context.Context, passID int) ([]Event, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, pass_id, event_type, ip, user_agent, attrs, created_at
		   FROM guest_pass_events
		  WHERE pass_id = $1
		  ORDER BY created_at DESC
		  LIMIT 200`, passID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.PassID, &e.Type, &e.IP, &e.UserAgent, &e.Attrs, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// PruneEvents deletes audit rows older than retentionDays. Called by the
// scheduled task. Returns the number of rows removed.
func (s *Store) PruneEvents(ctx context.Context, retentionDays int) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM guest_pass_events WHERE created_at < NOW() - ($1::text || ' days')::interval`,
		retentionDays)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
