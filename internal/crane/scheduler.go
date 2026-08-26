package crane

import (
	"context"
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
	if crane.ReservedKg+reservation.WeightKg > crane.CapacityKg {
		return domain.ErrCapacity
	}
	if err := s.store.SaveReservation(ctx, reservation); err != nil {
		return fmt.Errorf("save crane reservation: %w", err)
	}
	return nil
}
