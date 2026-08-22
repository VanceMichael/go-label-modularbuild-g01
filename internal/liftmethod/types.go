package liftmethod

import (
	"context"
	"time"
)

type Statement struct {
	ID        string
	TenantID  string
	Title     string
	Revision  string
	Version   int
	UpdatedBy string
	UpdatedAt time.Time
}

type Review interface {
	Check(context.Context, Statement) error
}

type ReviewFunc func(context.Context, Statement) error

func (f ReviewFunc) Check(ctx context.Context, statement Statement) error {
	return f(ctx, statement)
}

type Store interface {
	Add(Statement) error
	Get(context.Context, string, string) (Statement, error)
	Save(context.Context, Statement, int) error
	AuditCount() int
}
