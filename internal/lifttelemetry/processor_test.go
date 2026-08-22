package lifttelemetry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

// recordingSink blocks Publish on a per-event basis and records every
// published event in submit order.
type recordingSink struct {
	mu       sync.Mutex
	events   []Event
	gate     sync.Mutex
	gated    atomic.Bool
	gateCh   chan struct{}
	publishN atomic.Int32
}

func newRecordingSink() *recordingSink {
	return &recordingSink{gateCh: make(chan struct{})}
}

func (s *recordingSink) Publish(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.ID == "" || event.TenantID == "" || event.CraneID == "" || event.LoadKg <= 0 || event.ObservedAt.IsZero() {
		return domain.ErrInvalid
	}
	// Hold telemetry-1 inside Publish so the caller can start Shutdown while
	// telemetry-1 is still in flight and telemetry-2 is already queued.
	if s.gated.Load() && event.ID == "telemetry-1" {
		<-s.gateCh
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	s.publishN.Add(1)
	return nil
}

func (s *recordingSink) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Event(nil), s.events...)
}

func validEvent(id string, observedAt time.Time) Event {
	return Event{
		ID:         id,
		TenantID:   "tenant-1",
		CraneID:    "crane-23",
		LoadKg:     9000,
		ObservedAt: observedAt,
	}
}

// TestProcessorShutdownWaitsForInflightAndQueued reproduces the on-site
// incident ticket (telemetry-1, tenant-1, crane-23, telemetry-2):
//
//   - telemetry-1 is mid-Publish
//   - telemetry-2 has been accepted by Submit
//   - Shutdown is then started
//
// Shutdown must wait for both events to be delivered in submit order
// (telemetry-1 then telemetry-2) before returning, and a new Submit after
// Shutdown returns must yield domain.ErrState.
func TestProcessorShutdownWaitsForInflightAndQueued(t *testing.T) {
	sink := newRecordingSink()
	sink.gated.Store(true)

	p := NewProcessor(sink, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	base := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)

	// telemetry-1 enters Publish and blocks there (sink.gate is closed).
	if err := p.Submit(ctx, validEvent("telemetry-1", base)); err != nil {
		t.Fatalf("submit telemetry-1: %v", err)
	}
	// Wait until telemetry-1 is actually inside Publish.
	waitFor(t, func() bool { return sink.publishN.Load() == 0 }, 50*time.Millisecond, "telemetry-1 enter publish")

	// telemetry-2 is accepted (queued) while telemetry-1 is still publishing.
	if err := p.Submit(ctx, validEvent("telemetry-2", base.Add(time.Second))); err != nil {
		t.Fatalf("submit telemetry-2: %v", err)
	}

	// Start Shutdown on a separate goroutine: it must block until both
	// events are delivered.
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- p.Shutdown(ctx)
	}()

	// Confirm Shutdown has not returned yet while telemetry-1 is still
	// blocked inside Publish and telemetry-2 is still queued.
	select {
	case err := <-shutdownDone:
		t.Fatalf("shutdown returned before events delivered: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	// Release telemetry-1 so Publish returns and the run loop drains
	// telemetry-2 in order.
	close(sink.gateCh)

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not return after releasing telemetry-1")
	}

	// Both events delivered in submit order.
	got := sink.Events()
	if len(got) != 2 {
		t.Fatalf("expected 2 delivered events, got %d: %v", len(got), got)
	}
	if got[0].ID != "telemetry-1" || got[1].ID != "telemetry-2" {
		t.Fatalf("events delivered out of order: %+v", got)
	}

	// New Submit after Shutdown must yield ErrState (processor no longer
	// accepting).
	if err := p.Submit(ctx, validEvent("telemetry-3", base.Add(2*time.Second))); !errors.Is(err, domain.ErrState) {
		t.Fatalf("expected domain.ErrState after shutdown, got %v", err)
	}
}

// TestProcessorNormalSubmitDelivered covers the normal processing path:
// events submitted while accepting are published to the sink.
func TestProcessorNormalSubmitDelivered(t *testing.T) {
	sink := newRecordingSink()
	p := NewProcessor(sink, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	base := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if err := p.Submit(ctx, validEvent(fmt.Sprintf("telemetry-%d", i+1), base.Add(time.Duration(i)*time.Second))); err != nil {
			t.Fatalf("submit %d: %v", i+1, err)
		}
	}
	waitFor(t, func() bool { return len(sink.Events()) == 3 }, 200*time.Millisecond, "deliver 3 events")
	if err := p.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

// TestProcessorSubmitAfterShutdownRejects covers the rejected path: once
// Shutdown has returned, every subsequent Submit returns ErrState (processor
// no longer accepting).
func TestProcessorSubmitAfterShutdownRejects(t *testing.T) {
	sink := newRecordingSink()
	p := NewProcessor(sink, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	base := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	if err := p.Submit(ctx, validEvent("telemetry-1", base)); err != nil {
		t.Fatalf("submit telemetry-1: %v", err)
	}
	waitFor(t, func() bool { return len(sink.Events()) == 1 }, 200*time.Millisecond, "telemetry-1 delivered")

	if err := p.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	// After Shutdown returns, new submits must be rejected with ErrState.
	if err := p.Submit(ctx, validEvent("telemetry-2", base.Add(time.Second))); !errors.Is(err, domain.ErrState) {
		t.Fatalf("expected domain.ErrState after shutdown, got %v", err)
	}
	if err := p.Shutdown(ctx); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
}

// TestProcessorShutdownDrainsQueuedAfterPublishError covers the failed
// finalization + recovery path: telemetry-1's Publish fails while
// telemetry-2 was already accepted and queued. Shutdown must still drain
// telemetry-2 in order and complete, and a subsequent Submit must return
// ErrState.
func TestProcessorShutdownDrainsQueuedAfterPublishError(t *testing.T) {
	sink := &gatedFailingSink{release: make(chan struct{})}
	p := NewProcessor(sink, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	base := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	if err := p.Submit(ctx, validEvent("telemetry-1", base)); err != nil {
		t.Fatalf("submit telemetry-1: %v", err)
	}
	// telemetry-1 is mid-Publish and blocked; queue telemetry-2.
	if err := p.Submit(ctx, validEvent("telemetry-2", base.Add(time.Second))); err != nil {
		t.Fatalf("submit telemetry-2: %v", err)
	}

	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- p.Shutdown(ctx) }()
	// Shutdown must block while telemetry-1 is still inside Publish.
	select {
	case err := <-shutdownDone:
		t.Fatalf("shutdown returned during in-flight publish: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	// Release telemetry-1; its Publish fails (failed finalization), then the
	// run loop drains telemetry-2 (recovery).
	close(sink.release)
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not return after releasing telemetry-1")
	}

	got := sink.Events()
	if len(got) != 1 || got[0].ID != "telemetry-2" {
		t.Fatalf("expected telemetry-2 delivered after drain, got %+v", got)
	}
	if err := p.Submit(ctx, validEvent("telemetry-3", base.Add(2*time.Second))); !errors.Is(err, domain.ErrState) {
		t.Fatalf("expected domain.ErrState after shutdown, got %v", err)
	}
}

type gatedFailingSink struct {
	mu      sync.Mutex
	events  []Event
	release chan struct{}
}

func (s *gatedFailingSink) Publish(ctx context.Context, event Event) error {
	if event.ID == "telemetry-1" {
		select {
		case <-s.release:
		case <-ctx.Done():
			return ctx.Err()
		}
		// Failed finalization: telemetry-1's Publish returns an error.
		return fmt.Errorf("publish telemetry-1 failed")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *gatedFailingSink) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Event(nil), s.events...)
}

func waitFor(t *testing.T, cond func() bool, timeout time.Duration, what string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("condition not met: %s", what)
}
