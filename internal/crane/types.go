package crane

import "context"

type Crane struct {
	ID         string
	TenantID   string
	CapacityKg int64
	ReservedKg int64
}

type Reservation struct {
	ID           string
	TenantID     string
	CraneID      string
	ModuleMoveID string
	WeightKg     int64
}

type Store interface {
	GetCrane(context.Context, string, string) (Crane, error)
	SaveReservation(context.Context, Reservation) error
	Reservations(context.Context, string, string) ([]Reservation, error)
}
