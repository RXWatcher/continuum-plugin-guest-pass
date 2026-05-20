package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// AppConfig is the singleton operator-tunable config row.
type AppConfig struct {
	PublicBaseURL     string `json:"public_base_url"`
	AuditRetentionDay int    `json:"audit_retention_days"`
}

// DefaultAppConfig returns the values used when no row exists yet.
func DefaultAppConfig() AppConfig {
	return AppConfig{AuditRetentionDay: 180}
}

// GetAppConfig reads the singleton row, creating it lazily if missing.
func (s *Store) GetAppConfig(ctx context.Context) (AppConfig, error) {
	cfg := DefaultAppConfig()
	err := s.pool.QueryRow(ctx, `
SELECT public_base_url, audit_retention_days
FROM app_config WHERE id = 1`).Scan(&cfg.PublicBaseURL, &cfg.AuditRetentionDay)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := s.pool.Exec(ctx, `INSERT INTO app_config (id) VALUES (1) ON CONFLICT (id) DO NOTHING`); err != nil {
			return AppConfig{}, fmt.Errorf("ensure app_config: %w", err)
		}
		return s.GetAppConfig(ctx)
	}
	if err != nil {
		return AppConfig{}, fmt.Errorf("get app_config: %w", err)
	}
	return normalizeAppConfig(cfg), nil
}

// UpdateAppConfig upserts the singleton row.
func (s *Store) UpdateAppConfig(ctx context.Context, cfg AppConfig) error {
	cfg = normalizeAppConfig(cfg)
	_, err := s.pool.Exec(ctx, `
INSERT INTO app_config (id, public_base_url, audit_retention_days, updated_at)
VALUES (1, $1, $2, NOW())
ON CONFLICT (id) DO UPDATE SET
    public_base_url      = EXCLUDED.public_base_url,
    audit_retention_days = EXCLUDED.audit_retention_days,
    updated_at           = NOW()`, cfg.PublicBaseURL, cfg.AuditRetentionDay)
	if err != nil {
		return fmt.Errorf("update app_config: %w", err)
	}
	return nil
}

// ImportLegacyAppConfig seeds the singleton row from manifest-supplied
// values only when the DB row still holds defaults. Once an operator has
// edited the row through the API, manifest changes are ignored.
func (s *Store) ImportLegacyAppConfig(ctx context.Context, legacy AppConfig) (AppConfig, error) {
	current, err := s.GetAppConfig(ctx)
	if err != nil {
		return AppConfig{}, err
	}
	if current != DefaultAppConfig() {
		return current, nil
	}
	next := normalizeAppConfig(legacy)
	if next == current {
		return current, nil
	}
	if err := s.UpdateAppConfig(ctx, next); err != nil {
		return AppConfig{}, err
	}
	return s.GetAppConfig(ctx)
}

func normalizeAppConfig(cfg AppConfig) AppConfig {
	cfg.PublicBaseURL = strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/")
	if cfg.AuditRetentionDay < 1 {
		cfg.AuditRetentionDay = 180
	}
	return cfg
}
