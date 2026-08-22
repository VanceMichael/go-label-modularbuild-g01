package erectiondispatch

import (
	"context"
	"errors"
)

var (
	ErrInvalidJob = errors.New("erection dispatch: invalid job")
	ErrNoJob      = errors.New("erection dispatch: no job available")
)

type Job struct {
	ID       string
	TenantID string
	ModuleID string
	CraneID  string
}

type AdmissionGate interface {
	BeforeWait(context.Context) error
}

type Pulse interface {
	Signal()
	Wait(context.Context) error
}

type OpenGate struct{}

func (OpenGate) BeforeWait(context.Context) error { return nil }
