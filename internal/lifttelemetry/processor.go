package lifttelemetry

import (
	"context"
	"fmt"
	"sync"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Processor struct {
	sink Sink

	mu        sync.Mutex
	accepting bool
	started   bool
	queue     chan Event
	stopping  chan struct{}
	done      chan struct{}
	stopOnce  sync.Once
}

func NewProcessor(sink Sink, queueSize int) *Processor {
	if queueSize < 1 {
		queueSize = 1
	}
	return &Processor{
		sink:      sink,
		accepting: true,
		queue:     make(chan Event, queueSize),
		stopping:  make(chan struct{}),
		done:      make(chan struct{}),
	}
}

func (p *Processor) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.sink == nil || p.started {
		p.mu.Unlock()
		return domain.ErrState
	}
	p.started = true
	p.mu.Unlock()
	go p.run(ctx)
	return nil
}

func (p *Processor) Submit(ctx context.Context, event Event) error {
	p.mu.Lock()
	accepting := p.accepting
	p.mu.Unlock()
	if !accepting {
		return domain.ErrState
	}
	select {
	case p.queue <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-p.stopping:
		return domain.ErrState
	}
}

func (p *Processor) Stopping() <-chan struct{} {
	return p.stopping
}

func (p *Processor) Shutdown(ctx context.Context) error {
	p.stopOnce.Do(func() {
		p.mu.Lock()
		p.accepting = false
		p.mu.Unlock()
		close(p.stopping)
	})
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Processor) run(ctx context.Context) {
	defer close(p.done)
	for {
		// Once Shutdown has been requested, stop accepting work but keep
		// draining events that were already accepted (e.g. telemetry-1 in
		// flight and telemetry-2 already queued) so that both are delivered
		// in submit order before run exits.
		select {
		case <-p.stopping:
			p.drain(ctx)
			return
		default:
		}

		select {
		case <-ctx.Done():
			return
		case event := <-p.queue:
			if err := p.sink.Publish(ctx, event); err != nil {
				_ = fmt.Errorf("publish lift telemetry: %w", err)
			}
		case <-p.stopping:
			p.drain(ctx)
			return
		}
	}
}

// drain publishes every event that was already accepted into the queue
// before Shutdown closed stopping, preserving submit order. It returns once
// the queue is empty so that Shutdown can complete.
func (p *Processor) drain(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-p.queue:
			if !ok {
				return
			}
			if err := p.sink.Publish(ctx, event); err != nil {
				_ = fmt.Errorf("publish lift telemetry: %w", err)
			}
		default:
			return
		}
	}
}
