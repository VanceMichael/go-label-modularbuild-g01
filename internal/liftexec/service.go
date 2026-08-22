package liftexec

import (
	"context"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) WithClock(now func() time.Time) *Service {
	s.now = now
	return s
}

func (s *Service) Finalize(ctx context.Context, tenantID, executionID, actorID string) error {
	if s.store == nil || s.now == nil || tenantID == "" || executionID == "" || actorID == "" {
		return domain.ErrInvalid
	}
	return s.store.Transaction(ctx, func(tx Tx) error {
		execution, err := tx.GetExecution(ctx, tenantID, executionID)
		if err != nil {
			return fmt.Errorf("load lift execution: %w", err)
		}
		if execution.Status != ExecutionActive {
			return domain.ErrState
		}
		reservation, err := tx.GetReservation(ctx, tenantID, execution.ReservationID)
		if err != nil {
			return fmt.Errorf("load crane reservation: %w", err)
		}
		if reservation.Status != ReservationHeld {
			return domain.ErrState
		}

		completedAt := s.now()
		execution.Status = ExecutionCompleted
		execution.CompletedAt = &completedAt
		execution.Version++
		reservation.Status = ReservationReleased
		reservation.ReleasedAt = &completedAt
		reservation.Version++
		if err := tx.SaveExecution(ctx, execution); err != nil {
			return fmt.Errorf("save lift execution: %w", err)
		}
		if err := tx.SaveReservation(ctx, reservation); err != nil {
			return fmt.Errorf("release crane reservation: %w", err)
		}
		if err := tx.AppendAudit(ctx, AuditEvent{
			TenantID:    tenantID,
			ExecutionID: executionID,
			ActorID:     actorID,
			Action:      "lift_execution.completed",
			At:          completedAt,
		}); err != nil {
			return fmt.Errorf("append completion audit: %w", err)
		}
		return nil
	})
}
