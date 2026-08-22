package lifttelemetry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type synchronizedSink struct {
	started chan struct{}
	release chan struct{}
	seen    chan Event
	once    sync.Once
}

func (s *synchronizedSink) Publish(ctx context.Context, event Event) error {
	s.once.Do(func() { close(s.started) })
	if s.release != nil {
		select {
		case <-s.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.seen <- event
	return nil
}

func TestGracefulShutdownDrainsAcceptedTelemetry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 22, 18, 0, 0, 0, time.UTC)
	events := []Event{
		{ID: "telemetry-1", TenantID: "tenant-1", CraneID: "crane-23", LoadKg: 4100, ObservedAt: now},
		{ID: "telemetry-2", TenantID: "tenant-1", CraneID: "crane-23", LoadKg: 4300, ObservedAt: now.Add(time.Second)},
	}

	t.Run("shutdown drains events accepted before it started", func(t *testing.T) {
		sink := &synchronizedSink{started: make(chan struct{}), release: make(chan struct{}), seen: make(chan Event, 2)}
		processor := NewProcessor(sink, 2)
		if err := processor.Start(ctx); err != nil {
			t.Fatal(err)
		}
		if err := processor.Submit(ctx, events[0]); err != nil {
			t.Fatal(err)
		}
		<-sink.started
		if err := processor.Submit(ctx, events[1]); err != nil {
			t.Fatal(err)
		}
		shutdownResult := make(chan error, 1)
		go func() { shutdownResult <- processor.Shutdown(ctx) }()
		<-processor.Stopping()
		close(sink.release)
		if err := <-shutdownResult; err != nil {
			t.Fatalf("shutdown: %v", err)
		}
		close(sink.seen)
		published := make([]Event, 0, 2)
		for event := range sink.seen {
			published = append(published, event)
		}
		if len(published) != 2 || published[0] != events[0] || published[1] != events[1] {
			t.Errorf("published events after shutdown = %+v", published)
		}
		if err := processor.Submit(ctx, Event{ID: "late"}); !errors.Is(err, domain.ErrState) {
			t.Errorf("submit after shutdown error = %v, want invalid state", err)
		}
	})

	t.Run("shutdown after normal delivery preserves order", func(t *testing.T) {
		sink := &synchronizedSink{started: make(chan struct{}), seen: make(chan Event, 2)}
		processor := NewProcessor(sink, 2)
		if err := processor.Start(ctx); err != nil {
			t.Fatal(err)
		}
		for _, event := range events {
			if err := processor.Submit(ctx, event); err != nil {
				t.Fatal(err)
			}
		}
		first := <-sink.seen
		second := <-sink.seen
		if first != events[0] || second != events[1] {
			t.Errorf("normal published order = %+v, %+v", first, second)
		}
		if err := processor.Shutdown(ctx); err != nil {
			t.Fatalf("normal shutdown: %v", err)
		}
	})
}
