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
	stop      chan struct{}
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
		stop:      make(chan struct{}),
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
		close(p.stop)
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
		select {
		case <-ctx.Done():
			return
		case <-p.stop:
			return
		default:
		}

		select {
		case <-ctx.Done():
			return
		case <-p.stop:
			return
		case event := <-p.queue:
			if err := p.sink.Publish(ctx, event); err != nil {
				_ = fmt.Errorf("publish lift telemetry: %w", err)
			}
		}
	}
}
