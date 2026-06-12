package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Pass is the operator-facing representation of a guest pass. Hash fields are
// json:"-" so they never leak in API responses.
type Pass struct {
	ID                       int        `json:"id"`
	TokenHash                string     `json:"-"`
	PINHash                  string     `json:"-"`
	Title                    string     `json:"title"`
	TargetType               string     `json:"target_type"`
	TargetID                 string     `json:"target_id"`
	Note                     string     `json:"note,omitempty"`
	CreatedBy                string     `json:"created_by"`
	ExpiresAt                time.Time  `json:"expires_at"`
	ValidHoursAfterFirstOpen int        `json:"valid_hours_after_first_open"`
	MaxOpens                 int        `json:"max_opens"`
	MaxPlays                 int        `json:"max_plays"`
	MaxWatchMinutes          int        `json:"max_watch_minutes"`
	MaxConcurrentStreams     int        `json:"max_concurrent_streams"`
	MaxDevices               int        `json:"max_devices"`
	MaxResolution            string     `json:"max_resolution"`
	AllowDownloads           bool       `json:"allow_downloads"`
	AllowDirectPlay          bool       `json:"allow_direct_play"`
	LockToFirstIP            bool       `json:"lock_to_first_ip"`
	RequirePIN               bool       `json:"require_pin"`
	DisableSeeking           bool       `json:"disable_seeking"`
	WatermarkMode            string     `json:"watermark_mode"`
	WatermarkProfile         string     `json:"watermark_profile"`
	WatermarkLogoURL         string     `json:"watermark_logo_url"`
	IPAllowlist              []string   `json:"ip_allowlist"`
	CountryAllowlist         []string   `json:"country_allowlist"`
	SessionGraceMinutes      int        `json:"session_grace_minutes"`
	PerItemPlayCount         bool       `json:"per_item_play_count"`
	Geofence                 []string   `json:"geofence"`
	OpenCount                int        `json:"open_count"`
	PlayCount                int        `json:"play_count"`
	FirstIP                  string     `json:"first_ip,omitempty"`
	FirstOpenedAt            *time.Time `json:"first_opened_at,omitempty"`
	RevokedAt                *time.Time `json:"revoked_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// CreatePassInput is the validated payload from the admin create endpoint.
type CreatePassInput struct {
	Title                    string
	TargetType               string
	TargetID                 string
	Note                     string
	CreatedBy                string
	ExpiresAt                time.Time
	ValidHoursAfterFirstOpen int
	MaxOpens                 int
	MaxPlays                 int
	MaxWatchMinutes          int
	MaxConcurrentStreams     int
	MaxDevices               int
	MaxResolution            string
	AllowDownloads           bool
	AllowDirectPlay          bool
	LockToFirstIP            bool
	RequirePIN               bool
	PIN                      string
	DisableSeeking           bool
	WatermarkMode            string
	WatermarkProfile         string
	WatermarkLogoURL         string
	IPAllowlist              []string
	CountryAllowlist         []string
	SessionGraceMinutes      int
	PerItemPlayCount         bool
	Geofence                 []string
}

// CreatePass persists a new pass and returns the pass plus the plaintext token.
// The token is only available here — afterwards only the hash is stored.
func (s *Store) CreatePass(ctx context.Context, in CreatePassInput) (*Pass, string, error) {
	token, hash, err := GenerateToken()
	if err != nil {
		return nil, "", err
	}
	ipAllowlist, err := json.Marshal(in.IPAllowlist)
	if err != nil {
		return nil, "", err
	}
	countryAllowlist, err := json.Marshal(in.CountryAllowlist)
	if err != nil {
		return nil, "", err
	}
	geofence, err := json.Marshal(in.Geofence)
	if err != nil {
		return nil, "", err
	}
	pinHash := ""
	if in.RequirePIN {
		pinHash, err = HashPIN(in.PIN)
		if err != nil {
			return nil, "", err
		}
	}
	row := s.pool.QueryRow(ctx, `
INSERT INTO guest_passes (
    token_hash, title, target_type, target_id, note, created_by, expires_at,
    valid_hours_after_first_open, max_opens, max_plays, max_watch_minutes,
    max_concurrent_streams, max_devices, max_resolution, allow_downloads, allow_direct_play,
    lock_to_first_ip, require_pin, pin_hash, disable_seeking, watermark_mode, watermark_profile, watermark_logo_url,
    ip_allowlist, country_allowlist, session_grace_minutes, per_item_play_count, geofence
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28)
`+passReturningSQL(),
		hash, in.Title, in.TargetType, in.TargetID, in.Note, in.CreatedBy, in.ExpiresAt,
		in.ValidHoursAfterFirstOpen, in.MaxOpens, in.MaxPlays, in.MaxWatchMinutes,
		defaultPositive(in.MaxConcurrentStreams, 1), defaultPositive(in.MaxDevices, 1), in.MaxResolution,
		in.AllowDownloads, in.AllowDirectPlay, in.LockToFirstIP, in.RequirePIN, pinHash,
		in.DisableSeeking, in.WatermarkMode, in.WatermarkProfile, in.WatermarkLogoURL, ipAllowlist, countryAllowlist,
		in.SessionGraceMinutes, in.PerItemPlayCount, geofence)
	p, err := scanPass(row)
	if err != nil {
		return nil, "", err
	}
	return p, token, nil
}

// ListPasses returns the most-recent passes capped at limit (default 100, max 200).
func (s *Store) ListPasses(ctx context.Context, limit int) ([]Pass, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, passSelectSQL()+`
ORDER BY created_at DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Pass
	for rows.Next() {
		p, err := scanPass(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// GetPassByToken looks up a pass by its plaintext token. Returns ErrNotFound
// if no pass has that token's hash.
func (s *Store) GetPassByToken(ctx context.Context, token string) (*Pass, error) {
	hash := HashToken(token)
	row := s.pool.QueryRow(ctx, passSelectSQL()+` WHERE token_hash = $1`, hash)
	p, err := scanPass(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

// RevokePass marks a pass as revoked. Returns ErrNotFound if the pass is
// missing or already revoked.
func (s *Store) RevokePass(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `UPDATE guest_passes SET revoked_at = NOW(), updated_at = NOW() WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RecordOpen atomically increments the open counter, enforcing MaxOpens
// inside the UPDATE (no TOCTOU race). Returns ErrOpenLimit if the pass is
// already at its cap.
func (s *Store) RecordOpen(ctx context.Context, passID int, ip string) (*Pass, error) {
	row := s.pool.QueryRow(ctx, `
UPDATE guest_passes
SET open_count      = open_count + 1,
    first_ip        = CASE WHEN first_ip = '' THEN $2 ELSE first_ip END,
    first_opened_at = COALESCE(first_opened_at, NOW()),
    updated_at      = NOW()
WHERE id = $1
  AND revoked_at IS NULL
  AND (max_opens = 0 OR open_count < max_opens)
`+passReturningSQL(), passID, ip)
	p, err := scanPass(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOpenLimit
	}
	return p, err
}

// RecordPlay atomically increments the play counter, enforcing MaxPlays
// inside the UPDATE. Returns ErrPlayLimit if already at cap.
func (s *Store) RecordPlay(ctx context.Context, passID int) (*Pass, error) {
	row := s.pool.QueryRow(ctx, `
UPDATE guest_passes
SET play_count = play_count + 1, updated_at = NOW()
WHERE id = $1
  AND revoked_at IS NULL
  AND (max_plays = 0 OR play_count < max_plays)
`+passReturningSQL(), passID)
	p, err := scanPass(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlayLimit
	}
	return p, err
}

// DecrementPlay reverses a RecordPlay increment. Used to compensate when a
// play was counted but the downstream grant could not be produced (slot
// reservation or host mint failed), so a failed attempt does not burn a
// play against MaxPlays. Clamped at zero so it can never go negative.
func (s *Store) DecrementPlay(ctx context.Context, passID int) error {
	_, err := s.pool.Exec(ctx, `
UPDATE guest_passes
SET play_count = GREATEST(play_count - 1, 0), updated_at = NOW()
WHERE id = $1`, passID)
	return err
}

// RehashPIN replaces the stored PIN hash with a fresh bcrypt hash. Used by
// the verify path when a legacy sha256 hash matched, to upgrade in place.
func (s *Store) RehashPIN(ctx context.Context, passID int, pin string) error {
	hash, err := HashPIN(pin)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE guest_passes SET pin_hash = $2, updated_at = NOW() WHERE id = $1`, passID, hash)
	return err
}

// Pass methods are pure (no DB access) so they can be unit-tested without
// a database.

// PINMatch returns (matched, needsRehash). When needsRehash is true the
// caller should call Store.RehashPIN to upgrade legacy sha256 storage.
func (p Pass) PINMatch(pin string) (matched, needsRehash bool) {
	if !p.RequirePIN {
		return true, false
	}
	switch VerifyPIN(p.PINHash, pin) {
	case PINVerifyOK:
		return true, false
	case PINVerifyOKRehash:
		return true, true
	default:
		return false, false
	}
}

// AllowsIP applies the IP allowlist, which may contain bare addresses or
// CIDR ranges. Empty allowlist allows all.
func (p Pass) AllowsIP(ip string) bool {
	if len(p.IPAllowlist) == 0 {
		return true
	}
	parsed := net.ParseIP(ip)
	for _, allowed := range p.IPAllowlist {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if parsed != nil {
			if _, network, err := net.ParseCIDR(allowed); err == nil && network.Contains(parsed) {
				return true
			}
		}
		if allowed == ip {
			return true
		}
	}
	return false
}

// EffectiveExpiresAt is the earlier of expires_at and (first_opened_at +
// valid_hours_after_first_open). Captures the "expires N hours after the
// recipient first opens it" policy.
func (p Pass) EffectiveExpiresAt() time.Time {
	if p.FirstOpenedAt == nil || p.ValidHoursAfterFirstOpen <= 0 {
		return p.ExpiresAt
	}
	afterOpen := p.FirstOpenedAt.Add(time.Duration(p.ValidHoursAfterFirstOpen) * time.Hour)
	if afterOpen.Before(p.ExpiresAt) {
		return afterOpen
	}
	return p.ExpiresAt
}

// Status returns the human-readable state string surfaced to clients.
func (p Pass) Status(now time.Time) string {
	if p.RevokedAt != nil {
		return "revoked"
	}
	if !p.EffectiveExpiresAt().After(now) {
		return "expired"
	}
	if p.MaxOpens > 0 && p.OpenCount >= p.MaxOpens {
		return "open_limit_reached"
	}
	if p.MaxPlays > 0 && p.PlayCount >= p.MaxPlays {
		return "play_limit_reached"
	}
	return "active"
}

type passScanner interface {
	Scan(dest ...any) error
}

func scanPass(row passScanner) (*Pass, error) {
	var p Pass
	var ipAllowlist, countryAllowlist, geofence []byte
	err := row.Scan(&p.ID, &p.TokenHash, &p.PINHash, &p.Title, &p.TargetType, &p.TargetID, &p.Note, &p.CreatedBy, &p.ExpiresAt,
		&p.ValidHoursAfterFirstOpen, &p.MaxOpens, &p.MaxPlays, &p.MaxWatchMinutes,
		&p.MaxConcurrentStreams, &p.MaxDevices, &p.MaxResolution, &p.AllowDownloads, &p.AllowDirectPlay,
		&p.LockToFirstIP, &p.RequirePIN, &p.DisableSeeking, &p.WatermarkMode, &p.WatermarkProfile, &p.WatermarkLogoURL,
		&ipAllowlist, &countryAllowlist, &p.SessionGraceMinutes, &p.PerItemPlayCount, &geofence,
		&p.OpenCount, &p.PlayCount, &p.FirstIP, &p.FirstOpenedAt, &p.RevokedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	// A corrupt allowlist/geofence column must fail closed: if we cannot
	// decode the restriction we cannot prove the request is permitted, so
	// surface the error rather than silently dropping the restriction
	// (which would let any IP/country through).
	if err := json.Unmarshal(ipAllowlist, &p.IPAllowlist); err != nil {
		return nil, fmt.Errorf("decode ip_allowlist for pass %d: %w", p.ID, err)
	}
	if err := json.Unmarshal(countryAllowlist, &p.CountryAllowlist); err != nil {
		return nil, fmt.Errorf("decode country_allowlist for pass %d: %w", p.ID, err)
	}
	if err := json.Unmarshal(geofence, &p.Geofence); err != nil {
		return nil, fmt.Errorf("decode geofence for pass %d: %w", p.ID, err)
	}
	return &p, nil
}

const passColumns = `id, token_hash, pin_hash, title, target_type, target_id, note, created_by, expires_at,
    valid_hours_after_first_open, max_opens, max_plays, max_watch_minutes,
    max_concurrent_streams, max_devices, max_resolution, allow_downloads, allow_direct_play,
    lock_to_first_ip, require_pin, disable_seeking, watermark_mode, watermark_profile, watermark_logo_url,
    ip_allowlist, country_allowlist, session_grace_minutes, per_item_play_count, geofence,
    open_count, play_count, first_ip, first_opened_at, revoked_at, created_at, updated_at`

func passSelectSQL() string  { return `SELECT ` + passColumns + ` FROM guest_passes` }
func passReturningSQL() string { return `RETURNING ` + passColumns }
