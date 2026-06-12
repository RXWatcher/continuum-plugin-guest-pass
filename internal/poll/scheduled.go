package poll

import (
	"context"
	"errors"
	"sync/atomic"

	pluginv1 "github.com/Silo-Server/silo-plugin-sdk/pkg/pluginproto/silo/plugin/v1"

	"github.com/RXWatcher/silo-plugin-guest-pass/internal/store"
)

// expiredGrantGraceHours bounds how long an already-expired playback grant
// row lingers before the scheduled task reaps it. A small grace tolerates
// clock skew and in-flight diagnostics without letting dead rows accumulate.
const expiredGrantGraceHours = 1

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

	// Run each prune independently so a failure in one does not skip the
	// others; collect the errors and surface them together.
	var errs []error
	if _, err := st.PruneEvents(ctx, cfg.RetentionDays); err != nil {
		errs = append(errs, err)
	}
	if _, err := st.PruneExpiredGrants(ctx, expiredGrantGraceHours); err != nil {
		errs = append(errs, err)
	}
	if _, err := st.PruneStaleDevices(ctx, cfg.RetentionDays); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return &pluginv1.RunScheduledTaskResponse{}, nil
}
