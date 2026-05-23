package poll

import (
	"context"
	"sync/atomic"

	pluginv1 "github.com/ContinuumApp/continuum-plugin-sdk/pkg/pluginproto/silo/plugin/v1"

	"github.com/RXWatcher/silo-plugin-guest-pass/internal/store"
)

type Config struct {
	RetentionDays int
}

type Server struct {
	pluginv1.UnimplementedScheduledTaskServer
	store atomic.Pointer[store.Store]
	cfg   atomic.Pointer[Config]
}

func New() *Server { return &Server{} }

func (s *Server) Set(st *store.Store, cfg Config) {
	s.store.Store(st)
	s.cfg.Store(&cfg)
}

func (s *Server) Run(ctx context.Context, _ *pluginv1.RunScheduledTaskRequest) (*pluginv1.RunScheduledTaskResponse, error) {
	st := s.store.Load()
	cfg := s.cfg.Load()
	if st == nil || cfg == nil {
		return &pluginv1.RunScheduledTaskResponse{}, nil
	}
	_, err := st.PruneEvents(ctx, cfg.RetentionDays)
	if err != nil {
		return nil, err
	}
	return &pluginv1.RunScheduledTaskResponse{}, nil
}
