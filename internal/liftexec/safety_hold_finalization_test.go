package liftexec

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

func TestOpenSafetyHoldBlocksLiftFinalization(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 22, 16, 0, 0, 0, time.UTC)

	t.Run("open hold preserves execution and reservation", func(t *testing.T) {
		store := NewMemory()
		if err := store.AddReservation(Reservation{ID: "reservation-open", TenantID: "tenant-1", CraneID: "crane-4"}); err != nil {
			t.Fatal(err)
		}
		if err := store.AddExecution(Execution{ID: "execution-open", TenantID: "tenant-1", ModuleID: "module-20", ReservationID: "reservation-open"}); err != nil {
			t.Fatal(err)
		}
		if err := store.AddHold(SafetyHold{ID: "hold-open", TenantID: "tenant-1", ExecutionID: "execution-open", Reason: "structural inspection pending"}); err != nil {
			t.Fatal(err)
		}
		service := NewService(store).WithClock(func() time.Time { return now })
		err := service.Finalize(ctx, "tenant-1", "execution-open", "site-supervisor")
		if !errors.Is(err, domain.ErrState) {
			t.Errorf("finalize with open safety hold error = %v, want invalid state", err)
		}
		execution, err := store.GetExecution(ctx, "tenant-1", "execution-open")
		if err != nil || execution.Status != ExecutionActive || execution.Version != 1 || execution.CompletedAt != nil {
			t.Errorf("execution with open hold = %+v, err=%v", execution, err)
		}
		reservation, err := store.GetReservation(ctx, "tenant-1", "reservation-open")
		if err != nil || reservation.Status != ReservationHeld || reservation.Version != 1 || reservation.ReleasedAt != nil {
			t.Errorf("reservation with open hold = %+v, err=%v", reservation, err)
		}
		if count := store.AuditCount(); count != 0 {
			t.Errorf("audit count with open hold = %d, want 0", count)
		}
	})

	t.Run("resolved hold permits atomic finalization", func(t *testing.T) {
		store := NewMemory()
		if err := store.AddReservation(Reservation{ID: "reservation-resolved", TenantID: "tenant-1", CraneID: "crane-4"}); err != nil {
			t.Fatal(err)
		}
		if err := store.AddExecution(Execution{ID: "execution-resolved", TenantID: "tenant-1", ModuleID: "module-20", ReservationID: "reservation-resolved"}); err != nil {
			t.Fatal(err)
		}
		if err := store.AddHold(SafetyHold{ID: "hold-resolved", TenantID: "tenant-1", ExecutionID: "execution-resolved", Reason: "structural inspection pending"}); err != nil {
			t.Fatal(err)
		}
		if err := store.ResolveHold(ctx, "tenant-1", "hold-resolved"); err != nil {
			t.Fatal(err)
		}
		service := NewService(store).WithClock(func() time.Time { return now })
		if err := service.Finalize(ctx, "tenant-1", "execution-resolved", "site-supervisor"); err != nil {
			t.Fatalf("finalize after resolving safety hold: %v", err)
		}
		execution, err := store.GetExecution(ctx, "tenant-1", "execution-resolved")
		if err != nil || execution.Status != ExecutionCompleted || execution.Version != 2 || execution.CompletedAt == nil || !execution.CompletedAt.Equal(now) {
			t.Errorf("completed execution = %+v, err=%v", execution, err)
		}
		reservation, err := store.GetReservation(ctx, "tenant-1", "reservation-resolved")
		if err != nil || reservation.Status != ReservationReleased || reservation.Version != 2 || reservation.ReleasedAt == nil || !reservation.ReleasedAt.Equal(now) {
			t.Errorf("released reservation = %+v, err=%v", reservation, err)
		}
		if count := store.AuditCount(); count != 1 {
			t.Errorf("audit count after finalization = %d, want 1", count)
		}
	})
}
