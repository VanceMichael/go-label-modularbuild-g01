package worker

import (
	"context"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/repository"
	"log/slog"
	"sync"
	"time"
)

type OutboxRunner struct {
	repo      repository.OutboxRepository
	log       *slog.Logger
	publisher Publisher
	done      chan struct{}
	wg        sync.WaitGroup
}

type Publisher interface {
	Publish(context.Context, domain.OutboxEvent) error
}

type PublisherFunc func(context.Context, domain.OutboxEvent) error

func (f PublisherFunc) Publish(ctx context.Context, event domain.OutboxEvent) error {
	return f(ctx, event)
}

func NewOutboxRunner(repo repository.OutboxRepository, log *slog.Logger) *OutboxRunner {
	return &OutboxRunner{
		repo: repo,
		log:  log,
		publisher: PublisherFunc(func(ctx context.Context, _ domain.OutboxEvent) error {
			return ctx.Err()
		}),
		done: make(chan struct{}),
	}
}
func (r *OutboxRunner) WithPublisher(publisher Publisher) *OutboxRunner {
	r.publisher = publisher
	return r
}
func (r *OutboxRunner) Run(ctx context.Context) {
	r.wg.Add(1)
	defer r.wg.Done()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.done:
			return
		case now := <-ticker.C:
			r.tick(ctx, now)
		}
	}
}
func (r *OutboxRunner) Wait() { close(r.done); r.wg.Wait() }
func (r *OutboxRunner) tick(ctx context.Context, now time.Time) {
	events, err := r.repo.Claim(ctx, now, 20)
	if err != nil {
		r.log.Error("outbox claim failed", "error", err)
		return
	}
	for _, event := range events {
		if err := r.repo.MarkPublished(ctx, event.ID, now); err != nil {
			r.log.Error("outbox acknowledgement failed", "event_id", event.ID, "error", err)
			continue
		}
		if err := r.publisher.Publish(ctx, event); err != nil {
			_ = r.repo.MarkFailed(ctx, event.ID, now, err.Error())
			continue
		}
	}
}
