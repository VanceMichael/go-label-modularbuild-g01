package dispatch

import (
	"context"
	"time"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
)

type Job struct {
	ID           string
	TenantID     string
	ModuleMoveID string
	Status       Status
	Attempts     int
	LeaseUntil   *time.Time
}

type Store interface {
	ClaimPending(context.Context, string, time.Time, time.Duration) (Job, error)
	Complete(context.Context, string, int) error
	Requeue(context.Context, string, int) error
	Get(context.Context, string, string) (Job, error)
}

type Handler interface {
	Handle(context.Context, Job) error
}

type HandlerFunc func(context.Context, Job) error

func (f HandlerFunc) Handle(ctx context.Context, job Job) error {
	return f(ctx, job)
}
