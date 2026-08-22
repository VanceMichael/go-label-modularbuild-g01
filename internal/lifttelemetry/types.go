package lifttelemetry

import (
	"context"
	"time"
)

type Event struct {
	ID         string
	TenantID   string
	CraneID    string
	LoadKg     int64
	ObservedAt time.Time
}

type Sink interface {
	Publish(context.Context, Event) error
}
