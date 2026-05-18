package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound    = errors.New("guest pass not found")
	ErrDeviceLimit = errors.New("device limit reached")
)

type Store struct {
	pool *pgxpool.Pool
}

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

type Event struct {
	ID        int            `json:"id"`
	PassID    int            `json:"pass_id"`
	Type      string         `json:"type"`
	IP        string         `json:"ip,omitempty"`
	UserAgent string         `json:"user_agent,omitempty"`
	Attrs     map[string]any `json:"attrs,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type CatalogSearchItem struct {
	ContentID        string
	MediaFileID      int
	Type             string
	Title            string
	Year             int
	Overview         string
	PosterURL        string
	Genres           []string
	RuntimeMinutes   int
	ContentRating    string
	ExternalID       string
	ExternalProvider string
}

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

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS guest_passes (
	id BIGSERIAL PRIMARY KEY,
	token_hash TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	target_type TEXT NOT NULL,
	target_id TEXT NOT NULL,
	note TEXT NOT NULL DEFAULT '',
	created_by TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL,
	valid_hours_after_first_open INTEGER NOT NULL DEFAULT 0,
	max_opens INTEGER NOT NULL DEFAULT 0,
	max_plays INTEGER NOT NULL DEFAULT 0,
	max_watch_minutes INTEGER NOT NULL DEFAULT 0,
	max_concurrent_streams INTEGER NOT NULL DEFAULT 1,
	max_devices INTEGER NOT NULL DEFAULT 1,
	max_resolution TEXT NOT NULL DEFAULT '1080p',
	allow_downloads BOOLEAN NOT NULL DEFAULT FALSE,
	allow_direct_play BOOLEAN NOT NULL DEFAULT FALSE,
	lock_to_first_ip BOOLEAN NOT NULL DEFAULT FALSE,
	require_pin BOOLEAN NOT NULL DEFAULT FALSE,
	pin_hash TEXT NOT NULL DEFAULT '',
	disable_seeking BOOLEAN NOT NULL DEFAULT FALSE,
	watermark_profile TEXT NOT NULL DEFAULT '',
	watermark_mode TEXT NOT NULL DEFAULT 'none',
	watermark_logo_url TEXT NOT NULL DEFAULT '',
	ip_allowlist JSONB NOT NULL DEFAULT '[]'::jsonb,
	country_allowlist JSONB NOT NULL DEFAULT '[]'::jsonb,
	session_grace_minutes INTEGER NOT NULL DEFAULT 0,
	per_item_play_count BOOLEAN NOT NULL DEFAULT FALSE,
	geofence JSONB NOT NULL DEFAULT '[]'::jsonb,
	open_count INTEGER NOT NULL DEFAULT 0,
	play_count INTEGER NOT NULL DEFAULT 0,
	first_ip TEXT NOT NULL DEFAULT '',
	first_opened_at TIMESTAMPTZ,
	revoked_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS max_watch_minutes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS max_concurrent_streams INTEGER NOT NULL DEFAULT 1;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS max_devices INTEGER NOT NULL DEFAULT 1;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS max_resolution TEXT NOT NULL DEFAULT '1080p';
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS allow_downloads BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS allow_direct_play BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS lock_to_first_ip BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS require_pin BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS pin_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS disable_seeking BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS watermark_profile TEXT NOT NULL DEFAULT '';
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS watermark_mode TEXT NOT NULL DEFAULT 'none';
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS watermark_logo_url TEXT NOT NULL DEFAULT '';
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS ip_allowlist JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS country_allowlist JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS session_grace_minutes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS per_item_play_count BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS geofence JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE guest_passes ADD COLUMN IF NOT EXISTS first_ip TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS guest_passes_expires_at_idx ON guest_passes (expires_at);
CREATE INDEX IF NOT EXISTS guest_passes_target_idx ON guest_passes (target_type, target_id);
CREATE TABLE IF NOT EXISTS guest_pass_events (
	id BIGSERIAL PRIMARY KEY,
	pass_id BIGINT NOT NULL REFERENCES guest_passes(id) ON DELETE CASCADE,
	event_type TEXT NOT NULL,
	ip TEXT NOT NULL DEFAULT '',
	user_agent TEXT NOT NULL DEFAULT '',
	attrs JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS guest_pass_events_pass_id_idx ON guest_pass_events (pass_id, created_at DESC);
CREATE INDEX IF NOT EXISTS guest_pass_events_created_at_idx ON guest_pass_events (created_at);
CREATE TABLE IF NOT EXISTS guest_pass_devices (
	id BIGSERIAL PRIMARY KEY,
	pass_id BIGINT NOT NULL REFERENCES guest_passes(id) ON DELETE CASCADE,
	device_id TEXT NOT NULL,
	first_ip TEXT NOT NULL DEFAULT '',
	user_agent TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(pass_id, device_id)
);
CREATE TABLE IF NOT EXISTS guest_pass_grants (
	id BIGSERIAL PRIMARY KEY,
	pass_id BIGINT NOT NULL REFERENCES guest_passes(id) ON DELETE CASCADE,
	device_id TEXT NOT NULL DEFAULT '',
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS guest_pass_grants_active_idx ON guest_pass_grants (pass_id, expires_at);
`)
	return err
}

func GenerateToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func HashPIN(pin string) string {
	if pin == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("pin:" + pin))
	return hex.EncodeToString(sum[:])
}

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
		pinHash = HashPIN(in.PIN)
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

func (s *Store) ListPasses(ctx context.Context, limit int) ([]Pass, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
`+passSelectSQL()+`
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

func (s *Store) PlayableFileIDs(ctx context.Context, contentIDs []string) (map[string]int, error) {
	if len(contentIDs) == 0 {
		return map[string]int{}, nil
	}
	rows, err := s.pool.Query(ctx, `
WITH requested(content_id) AS (
	SELECT DISTINCT unnest($1::text[])
),
ranked AS (
	SELECT
		r.content_id,
		mf.id AS file_id,
		row_number() OVER (
			PARTITION BY r.content_id
			ORDER BY
				CASE lower(COALESCE(mf.resolution, ''))
					WHEN '2160p' THEN 1
					WHEN '4k' THEN 1
					WHEN '1080p' THEN 2
					WHEN '720p' THEN 3
					WHEN '480p' THEN 4
					ELSE 5
				END,
				mf.id
		) AS rn
	FROM requested r
	JOIN public.media_files mf ON (mf.content_id = r.content_id OR mf.episode_id = r.content_id)
	WHERE mf.missing_since IS NULL
)
SELECT content_id, file_id
FROM ranked
WHERE rn = 1`, contentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int, len(contentIDs))
	for rows.Next() {
		var contentID string
		var fileID int
		if err := rows.Scan(&contentID, &fileID); err != nil {
			return nil, err
		}
		out[contentID] = fileID
	}
	return out, rows.Err()
}

func (s *Store) SearchPlayableCatalog(ctx context.Context, query string, mediaTypes []string, limit int) ([]CatalogSearchItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if len(mediaTypes) == 0 {
		mediaTypes = []string{"movie", "episode"}
	}
	rows, err := s.pool.Query(ctx, `
WITH normalized AS (
	SELECT btrim(lower(regexp_replace($1, '[^[:alnum:]]+', ' ', 'g'))) AS q
)
SELECT
	mi.content_id,
	file_choice.id,
	mi.type,
	mi.title,
	COALESCE(mi.year, 0),
	COALESCE(mi.overview, ''),
	COALESCE(mi.poster_path, ''),
	COALESCE(mi.genres, '{}'::text[]),
	COALESCE(mi.runtime, 0),
	COALESCE(mi.content_rating, ''),
	COALESCE(NULLIF(mi.tmdb_id, ''), NULLIF(mi.tvdb_id, ''), NULLIF(mi.imdb_id, ''), ''),
	CASE
		WHEN NULLIF(mi.tmdb_id, '') IS NOT NULL THEN 'tmdb'
		WHEN NULLIF(mi.tvdb_id, '') IS NOT NULL THEN 'tvdb'
		WHEN NULLIF(mi.imdb_id, '') IS NOT NULL THEN 'imdb'
		ELSE ''
	END
FROM public.media_items mi
CROSS JOIN normalized n
JOIN LATERAL (
	SELECT mf.id
	FROM public.media_files mf
	WHERE mf.missing_since IS NULL
		AND (mf.content_id = mi.content_id OR mf.episode_id = mi.content_id)
	ORDER BY
		CASE lower(COALESCE(mf.resolution, ''))
			WHEN '2160p' THEN 1
			WHEN '4k' THEN 1
			WHEN '1080p' THEN 2
			WHEN '720p' THEN 3
			WHEN '480p' THEN 4
			ELSE 5
		END,
		mf.id
	LIMIT 1
) file_choice ON true
WHERE mi.type = ANY($2::text[])
	AND n.q <> ''
	AND mi.title_normalized LIKE '%' || n.q || '%'
ORDER BY
	CASE WHEN mi.title_normalized = n.q THEN 0 ELSE 1 END,
	CASE WHEN mi.title_normalized LIKE n.q || '%' THEN 0 ELSE 1 END,
	COALESCE(mi.year, 0) DESC,
	mi.title
LIMIT $3`, query, mediaTypes, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CatalogSearchItem, 0, limit)
	for rows.Next() {
		var item CatalogSearchItem
		if err := rows.Scan(
			&item.ContentID,
			&item.MediaFileID,
			&item.Type,
			&item.Title,
			&item.Year,
			&item.Overview,
			&item.PosterURL,
			&item.Genres,
			&item.RuntimeMinutes,
			&item.ContentRating,
			&item.ExternalID,
			&item.ExternalProvider,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetPassByToken(ctx context.Context, token string) (*Pass, error) {
	return s.getPassByHash(ctx, HashToken(token))
}

func (s *Store) getPassByHash(ctx context.Context, hash string) (*Pass, error) {
	row := s.pool.QueryRow(ctx, `
`+passSelectSQL()+` WHERE token_hash = $1`, hash)
	p, err := scanPass(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

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

func (s *Store) RecordOpen(ctx context.Context, passID int, ip, ua string) (*Pass, error) {
	row := s.pool.QueryRow(ctx, `
UPDATE guest_passes
SET open_count = open_count + 1,
	first_ip = CASE WHEN first_ip = '' THEN $2 ELSE first_ip END,
	first_opened_at = COALESCE(first_opened_at, NOW()),
	updated_at = NOW()
WHERE id = $1
`+passReturningSQL(), passID, ip)
	p, err := scanPass(row)
	if err != nil {
		return nil, err
	}
	_ = s.RecordEvent(ctx, passID, "opened", ip, ua, nil)
	return p, nil
}

func (s *Store) RecordPlay(ctx context.Context, passID int, ip, ua string) (*Pass, error) {
	row := s.pool.QueryRow(ctx, `
UPDATE guest_passes
SET play_count = play_count + 1, updated_at = NOW()
WHERE id = $1
`+passReturningSQL(), passID)
	p, err := scanPass(row)
	if err != nil {
		return nil, err
	}
	_ = s.RecordEvent(ctx, passID, "play_attempted", ip, ua, map[string]any{"grant": "unavailable"})
	return p, nil
}

func (s *Store) RegisterDevice(ctx context.Context, passID int, deviceID, ip, ua string, maxDevices int) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || maxDevices <= 0 {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM guest_pass_devices WHERE pass_id = $1 AND device_id = $2)`, passID, deviceID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		_, err = tx.Exec(ctx, `UPDATE guest_pass_devices SET last_seen_at = NOW(), first_ip = COALESCE(NULLIF(first_ip, ''), $3), user_agent = $4 WHERE pass_id = $1 AND device_id = $2`, passID, deviceID, ip, trimLen(ua, 512))
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM guest_pass_devices WHERE pass_id = $1`, passID).Scan(&count); err != nil {
		return err
	}
	if count >= maxDevices {
		return ErrDeviceLimit
	}
	_, err = tx.Exec(ctx, `INSERT INTO guest_pass_devices (pass_id, device_id, first_ip, user_agent) VALUES ($1,$2,$3,$4)`, passID, deviceID, ip, trimLen(ua, 512))
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ActiveGrantCount(ctx context.Context, passID int) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM guest_pass_grants WHERE pass_id = $1 AND expires_at > NOW()`, passID).Scan(&count)
	return count, err
}

func (s *Store) RecordGrant(ctx context.Context, passID int, deviceID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO guest_pass_grants (pass_id, device_id, expires_at) VALUES ($1,$2,$3)`, passID, trimLen(deviceID, 256), expiresAt)
	return err
}

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
	_, err = s.pool.Exec(ctx, `INSERT INTO guest_pass_events (pass_id, event_type, ip, user_agent, attrs) VALUES ($1,$2,$3,$4,$5)`, passID, typ, ip, ua, raw)
	return err
}

func trimLen(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func (s *Store) ListEvents(ctx context.Context, passID int) ([]Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, pass_id, event_type, ip, user_agent, attrs, created_at FROM guest_pass_events WHERE pass_id = $1 ORDER BY created_at DESC LIMIT 200`, passID)
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

func (s *Store) PruneEvents(ctx context.Context, retentionDays int) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM guest_pass_events WHERE created_at < NOW() - ($1::text || ' days')::interval`, retentionDays)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
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
	_ = json.Unmarshal(ipAllowlist, &p.IPAllowlist)
	_ = json.Unmarshal(countryAllowlist, &p.CountryAllowlist)
	_ = json.Unmarshal(geofence, &p.Geofence)
	return &p, nil
}

func passSelectSQL() string {
	return `SELECT id, token_hash, pin_hash, title, target_type, target_id, note, created_by, expires_at,
	valid_hours_after_first_open, max_opens, max_plays, max_watch_minutes,
	max_concurrent_streams, max_devices, max_resolution, allow_downloads, allow_direct_play,
	lock_to_first_ip, require_pin, disable_seeking, watermark_mode, watermark_profile, watermark_logo_url,
	ip_allowlist, country_allowlist, session_grace_minutes, per_item_play_count, geofence,
	open_count, play_count, first_ip, first_opened_at, revoked_at, created_at, updated_at
FROM guest_passes`
}

func passReturningSQL() string {
	return `RETURNING id, token_hash, pin_hash, title, target_type, target_id, note, created_by, expires_at,
	valid_hours_after_first_open, max_opens, max_plays, max_watch_minutes,
	max_concurrent_streams, max_devices, max_resolution, allow_downloads, allow_direct_play,
	lock_to_first_ip, require_pin, disable_seeking, watermark_mode, watermark_profile, watermark_logo_url,
	ip_allowlist, country_allowlist, session_grace_minutes, per_item_play_count, geofence,
	open_count, play_count, first_ip, first_opened_at, revoked_at, created_at, updated_at`
}

func defaultPositive(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func (p Pass) PINMatches(pin string) bool {
	if !p.RequirePIN {
		return true
	}
	return p.PINHash != "" && HashPIN(pin) == p.PINHash
}

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

func (p Pass) Status(now time.Time) string {
	if p.RevokedAt != nil {
		return "revoked"
	}
	if !p.EffectiveExpiresAt().After(now) {
		return "expired"
	}
	if p.MaxOpens > 0 && p.OpenCount > p.MaxOpens {
		return "open_limit_reached"
	}
	if p.MaxPlays > 0 && p.PlayCount >= p.MaxPlays {
		return "play_limit_reached"
	}
	return "active"
}
