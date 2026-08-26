package crane

import (
	"context"
	"errors"
	"fmt"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Scheduler struct {
	store Store
}

func NewScheduler(store Store) *Scheduler {
	return &Scheduler{store: store}
}

func (s *Scheduler) Reserve(ctx context.Context, reservation Reservation) error {
	if s.store == nil || reservation.ID == "" || reservation.TenantID == "" || reservation.CraneID == "" ||
		reservation.ModuleMoveID == "" || reservation.WeightKg <= 0 {
		return domain.ErrInvalid
	}
	crane, err := s.store.GetCrane(ctx, reservation.TenantID, reservation.CraneID)
	if err != nil {
		return fmt.Errorf("load crane: %w", err)
	}
	// Fast-path rejection against a possibly stale snapshot; the authoritative
	// capacity check happens atomically inside SaveReservation under the store
	// lock so that two concurrent reservations for the same crane cannot both
	// pass the check and persist. The store-level check is what guarantees at
	// most one of two competing requests succeeds.
	if crane.ReservedKg+reservation.WeightKg > crane.CapacityKg {
		return domain.ErrCapacity
	}
	if err := s.store.SaveReservation(ctx, reservation); err != nil {
		// A capacity rejection surfaced from the atomic store check is an
		// expected outcome under contention; surface it verbatim rather than
		// wrapping it as an opaque save failure.
		if errors.Is(err, domain.ErrCapacity) {
			return err
		}
		return fmt.Errorf("save crane reservation: %w", err)
	}
	return nil
}
