package lifttelemetry

import (
	"context"
	"sync"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type MemorySink struct {
	mu     sync.Mutex
	events []Event
}

func NewMemorySink() *MemorySink {
	return &MemorySink{}
}

func (s *MemorySink) Publish(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.ID == "" || event.TenantID == "" || event.CraneID == "" || event.LoadKg <= 0 || event.ObservedAt.IsZero() {
		return domain.ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *MemorySink) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Event(nil), s.events...)
}
