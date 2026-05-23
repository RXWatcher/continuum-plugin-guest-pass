package runtime

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	pluginv1 "github.com/ContinuumApp/continuum-plugin-sdk/pkg/pluginproto/silo/plugin/v1"
	"github.com/ContinuumApp/continuum-plugin-sdk/pkg/pluginsdk/runtimedefault"
)

type Config struct {
	DatabaseURL       string
	PublicBaseURL     string
	AuditRetentionDay int
}

type Server struct {
	runtimedefault.Server
	manifest *pluginv1.PluginManifest
	onConfig func(Config) error

	mu  sync.RWMutex
	cfg Config
}

func New(manifest *pluginv1.PluginManifest, onConfig func(Config) error) *Server {
	return &Server{manifest: manifest, onConfig: onConfig}
}

func (s *Server) GetManifest(context.Context, *pluginv1.GetManifestRequest) (*pluginv1.GetManifestResponse, error) {
	return &pluginv1.GetManifestResponse{Manifest: s.manifest}, nil
}

func (s *Server) Configure(_ context.Context, req *pluginv1.ConfigureRequest) (*pluginv1.ConfigureResponse, error) {
	cfg := Config{AuditRetentionDay: 180}
	for _, e := range req.GetConfig() {
		if e.GetValue() == nil {
			continue
		}
		m := e.GetValue().AsMap()
		switch e.GetKey() {
		case "database_url":
			cfg.DatabaseURL = stringValue(m["value"], firstString(m))
		case "public_base_url":
			cfg.PublicBaseURL = stringValue(m["value"], firstString(m))
		case "audit_retention_days":
			cfg.AuditRetentionDay = intValue(m["value"], firstNumber(m), cfg.AuditRetentionDay)
		}
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("database_url is required")
	}
	if cfg.PublicBaseURL != "" {
		u, err := url.Parse(cfg.PublicBaseURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("public_base_url must be an absolute URL")
		}
	}
	if cfg.AuditRetentionDay < 1 {
		cfg.AuditRetentionDay = 180
	}
	if s.onConfig != nil {
		if err := s.onConfig(cfg); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
	return &pluginv1.ConfigureResponse{}, nil
}

func stringValue(candidates ...any) string {
	for _, c := range candidates {
		if s, ok := c.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func intValue(a any, b any, fallback int) int {
	for _, c := range []any{a, b} {
		switch v := c.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return fallback
}

func firstString(m map[string]any) any {
	for _, v := range m {
		if _, ok := v.(string); ok {
			return v
		}
	}
	return nil
}

func firstNumber(m map[string]any) any {
	for _, v := range m {
		if _, ok := v.(float64); ok {
			return v
		}
	}
	return nil
}
