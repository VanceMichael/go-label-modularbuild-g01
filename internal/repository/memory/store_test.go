package memory

import (
	"context"
	"errors"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/repository"
	"testing"
	"time"
)

func TestMemoryUsersAndSessions(t *testing.T) {
	s := New()
	now := time.Now()
	u := domain.User{ID: "u", TenantID: "t", Email: "u@example.com", Active: true, Role: domain.RoleSitePlanner, CreatedAt: now}
	if err := s.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetUserByEmail(context.Background(), u.Email); err != nil {
		t.Fatal(err)
	}
	session := domain.Session{ID: "s", UserID: "u", TokenHash: "hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := s.CreateSession(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSession(context.Background(), "wrong"); err != domain.ErrNotFound {
		t.Fatal(err)
	}
	if err := s.DeactivateUser(context.Background(), "u", now); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetUser(context.Background(), "u")
	if got.Active {
		t.Fatal("user still active")
	}
}
func TestMemoryModuleMoveIdempotency(t *testing.T) {
	s := New()
	now := time.Now()
	v := domain.ModuleMove{ID: "s", TenantID: "t", Reference: "R", Origin: "PEK", Destination: "FRA", WeightKg: 1, Pieces: 1, Status: domain.ModuleMoveDraft, IdempotencyKey: "k", Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := s.CreateModuleMove(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	if _, err := s.FindByIdempotency(context.Background(), "t", "k"); err != nil {
		t.Fatal(err)
	}
	v.ID = "s2"
	if err := s.CreateModuleMove(context.Background(), v); err != domain.ErrConflict {
		t.Fatal(err)
	}
	page, err := s.ListModuleMoves(context.Background(), "t", domain.PageRequest{Limit: 10})
	if err != nil || page.Total != 1 {
		t.Fatal(err, page)
	}
}
func TestMemoryLegConcurrencyVersion(t *testing.T) {
	s := New()
	now := time.Now()
	l := domain.LiftWindow{ID: "l", TenantID: "t", LiftNumber: "F", Origin: "PEK", Destination: "FRA", DepartureAt: now.Add(time.Hour), ArrivalAt: now.Add(2 * time.Hour), CapacityKg: 10, Status: domain.WindowPlanned, Version: 1, CreatedAt: now}
	if err := s.CreateLeg(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	if err := s.ReserveCapacity(context.Background(), "t", "l", 5, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.ReserveCapacity(context.Background(), "t", "l", 5, 1); err != domain.ErrConflict {
		t.Fatal(err)
	}
	if err := s.ReserveCapacity(context.Background(), "t", "l", 6, 2); err != domain.ErrCapacity {
		t.Fatal(err)
	}
}

func TestMemoryBookingIsAtomicAndRouteScoped(t *testing.T) {
	s := New()
	now := time.Now().UTC()
	module_move := domain.ModuleMove{ID: "s", TenantID: "t", Reference: "R", Origin: "PEK", Destination: "FRA", WeightKg: 6, Pieces: 1, Status: domain.ModuleMoveDraft, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := s.CreateModuleMove(context.Background(), module_move); err != nil {
		t.Fatal(err)
	}
	window := domain.LiftWindow{ID: "l", TenantID: "t", LiftNumber: "AB1", Origin: "PEK", Destination: "FRA", DepartureAt: now.Add(time.Hour), ArrivalAt: now.Add(2 * time.Hour), CapacityKg: 10, Status: domain.WindowOpen, Version: 1, CreatedAt: now}
	if err := s.CreateLeg(context.Background(), window); err != nil {
		t.Fatal(err)
	}
	booked, err := s.BookModuleMove(context.Background(), "t", "s", "l", 1, 1, now)
	if err != nil || booked.Status != domain.ModuleMoveBooked {
		t.Fatalf("%v %#v", err, booked)
	}
	storedLeg, _ := s.GetLeg(context.Background(), "t", "l")
	if storedLeg.ReservedKg != 6 {
		t.Fatal(storedLeg.ReservedKg)
	}
}

func TestMemoryBookingRejectsRouteMismatchWithoutReservation(t *testing.T) {
	s := New()
	now := time.Now().UTC()
	_ = s.CreateModuleMove(context.Background(), domain.ModuleMove{ID: "s", TenantID: "t", Reference: "R", Origin: "PEK", Destination: "FRA", WeightKg: 6, Pieces: 1, Status: domain.ModuleMoveDraft, Version: 1, CreatedAt: now, UpdatedAt: now})
	_ = s.CreateLeg(context.Background(), domain.LiftWindow{ID: "l", TenantID: "t", LiftNumber: "AB1", Origin: "PVG", Destination: "FRA", DepartureAt: now.Add(time.Hour), ArrivalAt: now.Add(2 * time.Hour), CapacityKg: 10, Status: domain.WindowOpen, Version: 1, CreatedAt: now})
	if _, err := s.BookModuleMove(context.Background(), "t", "s", "l", 1, 1, now); err != domain.ErrInvalid {
		t.Fatal(err)
	}
	window, _ := s.GetLeg(context.Background(), "t", "l")
	if window.ReservedKg != 0 {
		t.Fatal("reservation leaked", window.ReservedKg)
	}
}

// TestFieldRecordMoveCancelReleasesWindow reproduces the field record where a
// booking cancelled after the lift capacity reservation leaked occupancy. On a
// cancelled booking the move stays draft with no window linkage and unchanged
// version, and the lift window must keep zero occupancy and its original
// version so that a retry or a concurrent caller proceeds on a clean slate.
func TestFieldRecordMoveCancelReleasesWindow(t *testing.T) {
	const tenant = "tenant-1"
	s := New()
	now := time.Now().UTC()
	move := domain.ModuleMove{ID: "MOD-A17", TenantID: tenant, Reference: "move-cancel", Origin: "FACTORY-A", Destination: "SITE-B", WeightKg: 6, Pieces: 1, Status: domain.ModuleMoveDraft, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := s.CreateModuleMove(context.Background(), move); err != nil {
		t.Fatal(err)
	}
	window := domain.LiftWindow{ID: "LIFT-08", TenantID: tenant, LiftNumber: "window-legal", Origin: "FACTORY-A", Destination: "SITE-B", DepartureAt: now.Add(time.Hour), ArrivalAt: now.Add(2 * time.Hour), CapacityKg: 10, Status: domain.WindowOpen, Version: 1, CreatedAt: now}
	if err := s.CreateLeg(context.Background(), window); err != nil {
		t.Fatal(err)
	}

	// Cancel arrives after the reservation inside AssignModuleMove. The memory
	// ReserveCapacity is synchronous and cannot observe a mid-call cancel, so
	// drive the same compensation path through the cancel branch directly.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	booked, err := repository.AssignModuleMove(ctx, s, tenant, move, window, now)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if booked.ID != "" {
		t.Fatalf("cancelled call must return empty move, got %s", booked.ID)
	}

	storedMove, _ := s.GetModuleMove(context.Background(), tenant, move.ID)
	if storedMove.Status != domain.ModuleMoveDraft {
		t.Fatalf("stored move status %s", storedMove.Status)
	}
	if storedMove.LegID != nil {
		t.Fatalf("window linkage leaked: %v", storedMove.LegID)
	}
	if storedMove.Version != 1 {
		t.Fatalf("move version changed: %d", storedMove.Version)
	}

	storedWindow, _ := s.GetLeg(context.Background(), tenant, window.ID)
	if storedWindow.ReservedKg != 0 {
		t.Fatalf("window occupancy leaked: %d", storedWindow.ReservedKg)
	}
	if storedWindow.Version != 1 {
		t.Fatalf("window version changed: %d", storedWindow.Version)
	}

	// Retry availability + success receipt: a clean-slate retry now succeeds and
	// reserves the module weight exactly once.
	retryMove, _ := s.GetModuleMove(context.Background(), tenant, move.ID)
	retryWindow, _ := s.GetLeg(context.Background(), tenant, window.ID)
	bookedRetry, err := repository.AssignModuleMove(context.Background(), s, tenant, retryMove, retryWindow, now)
	if err != nil || bookedRetry.Status != domain.ModuleMoveBooked {
		t.Fatalf("retry booking: %v %#v", err, bookedRetry)
	}
	storedWindowAfter, _ := s.GetLeg(context.Background(), tenant, window.ID)
	if storedWindowAfter.ReservedKg != move.WeightKg {
		t.Fatalf("window occupancy after retry: %d", storedWindowAfter.ReservedKg)
	}
	if storedWindowAfter.Version != 2 {
		t.Fatalf("window version after retry: %d", storedWindowAfter.Version)
	}
}
