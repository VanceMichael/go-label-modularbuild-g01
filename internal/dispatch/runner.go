package dispatch

import (
	"context"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Runner struct {
	store   Store
	handler Handler
	lease   time.Duration
	now     func() time.Time
}

func NewRunner(store Store, handler Handler, lease time.Duration) *Runner {
	return &Runner{store: store, handler: handler, lease: lease, now: time.Now}
}

func (r *Runner) WithClock(now func() time.Time) *Runner {
	r.now = now
	return r
}

func (r *Runner) RunOne(ctx context.Context, tenantID string) error {
	if r.store == nil || r.handler == nil || r.lease <= 0 || r.now == nil || tenantID == "" {
		return domain.ErrInvalid
	}
	job, err := r.store.ClaimPending(ctx, tenantID, r.now(), r.lease)
	if err != nil {
		return fmt.Errorf("claim dispatch job: %w", err)
	}
	if err := r.handler.Handle(ctx, job); err != nil {
		return fmt.Errorf("handle dispatch job: %w", err)
	}
	if err := r.store.Complete(ctx, job.ID, job.Attempts); err != nil {
		return fmt.Errorf("complete dispatch job: %w", err)
	}
	return nil
}
