package permit

import (
	"context"
	"errors"
	"time"
)

var ErrAuditUnavailable = errors.New("permit audit unavailable")

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
)

type Permit struct {
	ID         string
	TenantID   string
	ModuleID   string
	Status     Status
	ApprovedBy string
	ApprovedAt *time.Time
	Version    int
}

type AuditEvent struct {
	PermitID string
	TenantID string
	ActorID  string
	Action   string
	At       time.Time
}

type Tx interface {
	GetPermit(context.Context, string, string) (Permit, error)
	SavePermit(context.Context, Permit) error
	AppendAudit(context.Context, AuditEvent) error
}

type Store interface {
	Transaction(context.Context, func(Tx) error) error
	GetPermit(context.Context, string, string) (Permit, error)
}

type Cache interface {
	Put(context.Context, Permit) error
	Get(context.Context, string, string) (Permit, error)
}
