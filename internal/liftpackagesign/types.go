package liftpackagesign

import (
	"bytes"
	"context"
	"errors"
	"time"
)

var ErrInvalidPackage = errors.New("lift package signing: invalid package")

type Package struct {
	TenantID string    `json:"tenant_id"`
	PlanID   string    `json:"plan_id"`
	Modules  []string  `json:"modules"`
	IssuedAt time.Time `json:"issued_at"`
}

type SignedArtifact struct {
	PlanID   string
	Payload  []byte
	SignerID string
}

type Pending interface {
	Wait(context.Context) (SignedArtifact, error)
}

type Signer interface {
	Start(context.Context, string, []byte) (Pending, error)
}

type BufferPool interface {
	Acquire(context.Context) (*bytes.Buffer, error)
	Release(*bytes.Buffer)
}
