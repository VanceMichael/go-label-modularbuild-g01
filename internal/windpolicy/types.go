package windpolicy

import (
	"context"
	"time"
)

type Policy struct {
	TenantID      string
	MaxWindMPS    int
	EffectiveFrom time.Time
}

type Evaluation struct {
	Allowed     bool
	ObservedMPS int
	LimitMPS    int
}

type Loader interface {
	LoadActive(context.Context, string) (Policy, error)
}
