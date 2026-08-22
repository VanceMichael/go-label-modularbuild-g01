package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/repository"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/repository/memory"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/service"
)

type cancellationAfterCapacityStore struct {
	repository.Store
	cancel context.CancelFunc
}

func (s cancellationAfterCapacityStore) ReserveCapacity(
	ctx context.Context,
	tenant string,
	windowID string,
	weight int64,
	version int64,
) error {
	err := s.Store.ReserveCapacity(ctx, tenant, windowID, weight, version)
	if err == nil {
		s.cancel()
	}
	return err
}

func TestCancelledAssignmentPreservesMovementAndCapacity(t *testing.T) {
	now := time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)
	actor := domain.User{ID: "planner-1", TenantID: "tenant-1", Role: domain.RoleSitePlanner, Active: true}

	store := memory.New()
	seedAssignmentState(t, store, now, "move-cancel", "window-cancel")
	ctx, cancel := context.WithCancel(context.Background())
	app := service.NewApplication(
		cancellationAfterCapacityStore{Store: store, cancel: cancel},
		time.Hour,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	).WithClock(domain.FixedClock{T: now})

	if _, err := app.BookModuleMove(ctx, actor, "move-cancel", "window-cancel"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled assignment error = %v, want context cancellation", err)
	}
	move, moveErr := store.GetModuleMove(context.Background(), "tenant-1", "move-cancel")
	window, windowErr := store.GetLeg(context.Background(), "tenant-1", "window-cancel")
	if moveErr != nil || windowErr != nil {
		t.Fatalf("read cancelled assignment state: move=%v window=%v", moveErr, windowErr)
	}
	if move.Status != domain.ModuleMoveDraft || move.LegID != nil || move.Version != 1 {
		t.Errorf("cancelled assignment changed movement: %+v", move)
	}
	if window.ReservedKg != 0 || window.Version != 1 {
		t.Errorf("cancelled assignment changed capacity: %+v", window)
	}

	legalStore := memory.New()
	seedAssignmentState(t, legalStore, now, "move-legal", "window-legal")
	legalApp := service.NewApplication(
		legalStore,
		time.Hour,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	).WithClock(domain.FixedClock{T: now})
	result, err := legalApp.BookModuleMove(context.Background(), actor, "move-legal", "window-legal")
	if err != nil || result.Status != domain.ModuleMoveBooked || result.LegID == nil {
		t.Fatalf("legal assignment failed: result=%+v err=%v", result, err)
	}
	legalWindow, err := legalStore.GetLeg(context.Background(), "tenant-1", "window-legal")
	if err != nil || legalWindow.ReservedKg != 6 || legalWindow.Version != 2 {
		t.Fatalf("legal assignment capacity = %+v, err=%v", legalWindow, err)
	}
}

func seedAssignmentState(t *testing.T, store *memory.Store, now time.Time, moveID, windowID string) {
	t.Helper()
	move := domain.ModuleMove{ID: moveID, TenantID: "tenant-1", Reference: "MOD-A17", Origin: "FACTORY-A", Destination: "SITE-B", WeightKg: 6, Pieces: 1, Status: domain.ModuleMoveDraft, Version: 1, CreatedAt: now, UpdatedAt: now}
	window := domain.LiftWindow{ID: windowID, TenantID: "tenant-1", LiftNumber: "LIFT-08", Origin: "FACTORY-A", Destination: "SITE-B", DepartureAt: now.Add(time.Hour), ArrivalAt: now.Add(2 * time.Hour), CapacityKg: 20, Status: domain.WindowOpen, Version: 1, CreatedAt: now}
	if err := store.CreateModuleMove(context.Background(), move); err != nil {
		t.Fatalf("create movement: %v", err)
	}
	if err := store.CreateLeg(context.Background(), window); err != nil {
		t.Fatalf("create lift window: %v", err)
	}
}
