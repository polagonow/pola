package cron

import (
	"context"
	"sync"

	"github.com/polagonow/pola/core"
	robfigcron "github.com/robfig/cron/v3"
)

type Scheduler struct {
	mu   sync.Mutex
	cron *robfigcron.Cron
}

func New() *Scheduler {
	return &Scheduler{
		cron: robfigcron.New(robfigcron.WithSeconds()),
	}
}

func (s *Scheduler) Name() string { return "cron" }

func (s *Scheduler) AddFunc(spec string, fn func(ctx context.Context)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.cron.AddFunc(spec, func() { fn(context.Background()) })
	return err
}

func (s *Scheduler) Start(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cron.Stop()
	return nil
}

var _ core.TaskScheduler = (*Scheduler)(nil)
