package leasereaper

import (
	"context"
	"errors"
	"time"
)

var ErrLeaseMissing = errors.New("lease reaper: lease missing")

type Lease struct {
	TenantID string
	Resource string
	Owner    string
	Version  int64
	Expires  time.Time
}

type Store interface {
	Expired(context.Context, time.Time) ([]Lease, error)
	Delete(context.Context, string, string) error
	Renew(context.Context, string, string, time.Time) (Lease, error)
}

type Barrier interface {
	AfterScan(context.Context) error
}
