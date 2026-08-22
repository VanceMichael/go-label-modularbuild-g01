package safetyalarm

import (
	"context"
	"time"
)

type Alarm struct {
	ID         string
	TenantID   string
	CraneID    string
	Kind       string
	ObservedAt time.Time
}

type Handler interface {
	Handle(context.Context, Alarm) error
}

type HandlerFunc func(context.Context, Alarm) error

func (f HandlerFunc) Handle(ctx context.Context, alarm Alarm) error {
	return f(ctx, alarm)
}
