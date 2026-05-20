-- Guest passes: short-lived access tokens for scoped media playback.
CREATE TABLE IF NOT EXISTS guest_passes (
    id                            BIGSERIAL PRIMARY KEY,
    token_hash                    TEXT NOT NULL UNIQUE,
    title                         TEXT NOT NULL,
    target_type                   TEXT NOT NULL,
    target_id                     TEXT NOT NULL,
    note                          TEXT NOT NULL DEFAULT '',
    created_by                    TEXT NOT NULL,
    expires_at                    TIMESTAMPTZ NOT NULL,
    valid_hours_after_first_open  INTEGER NOT NULL DEFAULT 0,
    max_opens                     INTEGER NOT NULL DEFAULT 0,
    max_plays                     INTEGER NOT NULL DEFAULT 0,
    max_watch_minutes             INTEGER NOT NULL DEFAULT 0,
    max_concurrent_streams        INTEGER NOT NULL DEFAULT 1,
    max_devices                   INTEGER NOT NULL DEFAULT 1,
    max_resolution                TEXT NOT NULL DEFAULT '1080p',
    allow_downloads               BOOLEAN NOT NULL DEFAULT FALSE,
    allow_direct_play             BOOLEAN NOT NULL DEFAULT FALSE,
    lock_to_first_ip              BOOLEAN NOT NULL DEFAULT FALSE,
    require_pin                   BOOLEAN NOT NULL DEFAULT FALSE,
    pin_hash                      TEXT NOT NULL DEFAULT '',
    disable_seeking               BOOLEAN NOT NULL DEFAULT FALSE,
    watermark_mode                TEXT NOT NULL DEFAULT 'none',
    watermark_profile             TEXT NOT NULL DEFAULT '',
    watermark_logo_url            TEXT NOT NULL DEFAULT '',
    ip_allowlist                  JSONB NOT NULL DEFAULT '[]'::jsonb,
    country_allowlist             JSONB NOT NULL DEFAULT '[]'::jsonb,
    session_grace_minutes         INTEGER NOT NULL DEFAULT 0,
    per_item_play_count           BOOLEAN NOT NULL DEFAULT FALSE,
    geofence                      JSONB NOT NULL DEFAULT '[]'::jsonb,
    open_count                    INTEGER NOT NULL DEFAULT 0,
    play_count                    INTEGER NOT NULL DEFAULT 0,
    first_ip                      TEXT NOT NULL DEFAULT '',
    first_opened_at               TIMESTAMPTZ,
    revoked_at                    TIMESTAMPTZ,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS guest_passes_expires_at_idx ON guest_passes (expires_at);
CREATE INDEX IF NOT EXISTS guest_passes_target_idx     ON guest_passes (target_type, target_id);

-- Audit trail. attrs is free-form JSON for event-specific context.
CREATE TABLE IF NOT EXISTS guest_pass_events (
    id          BIGSERIAL PRIMARY KEY,
    pass_id     BIGINT NOT NULL REFERENCES guest_passes(id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL,
    ip          TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    attrs       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS guest_pass_events_pass_id_idx     ON guest_pass_events (pass_id, created_at DESC);
CREATE INDEX IF NOT EXISTS guest_pass_events_created_at_idx  ON guest_pass_events (created_at);

-- One row per (pass, device) so MaxDevices can be enforced.
CREATE TABLE IF NOT EXISTS guest_pass_devices (
    id            BIGSERIAL PRIMARY KEY,
    pass_id       BIGINT NOT NULL REFERENCES guest_passes(id) ON DELETE CASCADE,
    device_id     TEXT NOT NULL,
    first_ip      TEXT NOT NULL DEFAULT '',
    user_agent    TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(pass_id, device_id)
);

-- One row per outstanding playback grant so MaxConcurrentStreams can be enforced.
CREATE TABLE IF NOT EXISTS guest_pass_grants (
    id          BIGSERIAL PRIMARY KEY,
    pass_id     BIGINT NOT NULL REFERENCES guest_passes(id) ON DELETE CASCADE,
    device_id   TEXT NOT NULL DEFAULT '',
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS guest_pass_grants_active_idx ON guest_pass_grants (pass_id, expires_at);

-- Singleton config row.
CREATE TABLE IF NOT EXISTS app_config (
    id                    INTEGER PRIMARY KEY DEFAULT 1,
    public_base_url       TEXT NOT NULL DEFAULT '',
    audit_retention_days  INTEGER NOT NULL DEFAULT 180,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT app_config_singleton CHECK (id = 1)
);
INSERT INTO app_config (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
