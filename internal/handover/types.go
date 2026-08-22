package handover

import (
	"context"
	"errors"
	"time"
)

var ErrCertificationRejected = errors.New("handover: certification rejected")

type ModuleStatus string

const (
	ModuleStaged     ModuleStatus = "staged"
	ModuleHandedOver ModuleStatus = "handed_over"
)

type BatchStatus string

const (
	BatchPending   BatchStatus = "pending"
	BatchCompleted BatchStatus = "completed"
)

type Module struct {
	ID       string
	BatchID  string
	Status   ModuleStatus
	Revision int
}

type Batch struct {
	ID          string
	TenantID    string
	ModuleIDs   []string
	Status      BatchStatus
	CompletedAt time.Time
}

type AuditEntry struct {
	BatchID  string
	ModuleID string
	Action   string
	At       time.Time
}

type CertificationChecker interface {
	Check(context.Context, Module) error
}
