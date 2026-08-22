package cranevalidation

import (
	"context"
	"errors"
)

var ErrLeaseHeld = errors.New("crane validation: lease held")

type Request struct{ TenantID, CraneID, ConfigVersion string }
type Result struct {
	CraneID string
	Valid   bool
}
type Validator interface {
	Validate(context.Context, Request) (Result, error)
}
