package installationreceipt

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInstallationMissing = errors.New("installation receipt: installation missing")
	ErrInstallationState   = errors.New("installation receipt: invalid state")
)

type Installation struct {
	ID        string
	TenantID  string
	Status    string
	Version   int64
	ProofRef  string
	UpdatedAt time.Time
}

type Store interface {
	Get(context.Context, string, string) (Installation, error)
	Complete(context.Context, string, string, int64, string, time.Time) (Installation, error)
}

type Archive interface {
	Put(context.Context, string, []byte) (string, error)
}
