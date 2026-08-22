package readinessreview

import (
	"context"
	"errors"
	"time"
)

var ErrInspectionRejected = errors.New("readiness review: inspection rejected")

type Result struct {
	TenantID string
	BatchID  string
	ModuleID string
	Status   string
	Checked  time.Time
}

type Inspector interface {
	Inspect(context.Context, string) error
}

type Recorder interface {
	Record(context.Context, Result) error
}
