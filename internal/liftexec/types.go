package liftexec

import (
	"context"
	"time"
)

type ExecutionStatus string
type ReservationStatus string
type HoldStatus string

const (
	ExecutionActive    ExecutionStatus = "active"
	ExecutionCompleted ExecutionStatus = "completed"

	ReservationHeld     ReservationStatus = "held"
	ReservationReleased ReservationStatus = "released"

	HoldOpen     HoldStatus = "open"
	HoldResolved HoldStatus = "resolved"
)

type Execution struct {
	ID            string
	TenantID      string
	ModuleID      string
	ReservationID string
	Status        ExecutionStatus
	CompletedAt   *time.Time
	Version       int
}

type Reservation struct {
	ID         string
	TenantID   string
	CraneID    string
	Status     ReservationStatus
	ReleasedAt *time.Time
	Version    int
}

type SafetyHold struct {
	ID          string
	TenantID    string
	ExecutionID string
	Reason      string
	Status      HoldStatus
}

type AuditEvent struct {
	TenantID    string
	ExecutionID string
	ActorID     string
	Action      string
	At          time.Time
}

type Tx interface {
	GetExecution(context.Context, string, string) (Execution, error)
	GetReservation(context.Context, string, string) (Reservation, error)
	OpenHolds(context.Context, string, string) ([]SafetyHold, error)
	SaveExecution(context.Context, Execution) error
	SaveReservation(context.Context, Reservation) error
	AppendAudit(context.Context, AuditEvent) error
}

type Store interface {
	Transaction(context.Context, func(Tx) error) error
	GetExecution(context.Context, string, string) (Execution, error)
	GetReservation(context.Context, string, string) (Reservation, error)
}
