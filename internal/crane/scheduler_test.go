package crane

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

// TestConcurrent7kgReservationsCompeteOn10kgCrane reproduces the field flow
// captured by the business tags crane-17, tenant-1, reservation-a, module-a,
// reservation-b, module-b, crane-legal, reservation-4, module-4,
// reservation-6 and module-6: two synchronized competing 7kg reservation
// requests against a 10kg crane must end with exactly one success and one
// domain.ErrCapacity, the crane must end reserved at 7kg, and exactly one
// reservation must be persisted.
func TestConcurrent7kgReservationsCompeteOn10kgCrane(t *testing.T) {
	const (
		tenantID      = "tenant-1"
		craneID       = "crane-17"
		reservationA  = "reservation-a"
		moduleA       = "module-a"
		reservationB  = "reservation-b"
		moduleB       = "module-b"
		weightKg      = int64(7)
		capacityKg    = int64(10)
		goroutines    = 2
	)

	store := NewMemory()
	if err := store.AddCrane(Crane{
		ID:         craneID,
		TenantID:   tenantID,
		CapacityKg: capacityKg,
		ReservedKg: 0,
	}); err != nil {
		t.Fatalf("add crane: %v", err)
	}
	scheduler := NewScheduler(store)

	type result struct {
		err error
	}
	results := make(chan result, goroutines)
	var ready sync.WaitGroup
	var start sync.WaitGroup
	ready.Add(goroutines)
	start.Add(1)

	reservations := []Reservation{
		{ID: reservationA, TenantID: tenantID, CraneID: craneID, ModuleMoveID: moduleA, WeightKg: weightKg},
		{ID: reservationB, TenantID: tenantID, CraneID: craneID, ModuleMoveID: moduleB, WeightKg: weightKg},
	}

	for _, r := range reservations {
		r := r
		go func() {
			ready.Done()
			start.Wait()
			err := scheduler.Reserve(context.Background(), r)
			results <- result{err: err}
		}()
	}

	// Wait until both goroutines are parked at the start barrier, then release
	// them together to maximize the chance of overlapping execution and so
	// exercising the atomic capacity check.
	ready.Wait()
	start.Done()

	var successes, capacityErrors int
	for i := 0; i < goroutines; i++ {
		res := <-results
		switch {
		case res.err == nil:
			successes++
		case errors.Is(res.err, domain.ErrCapacity):
			capacityErrors++
		default:
			t.Fatalf("unexpected error: %v", res.err)
		}
	}

	if successes != 1 || capacityErrors != 1 {
		t.Fatalf("want exactly one success and one ErrCapacity, got %d success(es) and %d capacity error(s)", successes, capacityErrors)
	}

	// Re-read the persisted crane to assert the final reserved weight.
	stored, err := store.GetCrane(context.Background(), tenantID, craneID)
	if err != nil {
		t.Fatalf("get crane: %v", err)
	}
	if stored.ReservedKg != weightKg {
		t.Fatalf("crane reserved kg = %d, want %d", stored.ReservedKg, weightKg)
	}

	// Exactly one reservation must be persisted for this crane/tenant.
	persisted, err := store.Reservations(context.Background(), tenantID, craneID)
	if err != nil {
		t.Fatalf("list reservations: %v", err)
	}
	if len(persisted) != 1 {
		t.Fatalf("persisted reservations = %d, want 1", len(persisted))
	}
	if persisted[0].WeightKg != weightKg {
		t.Fatalf("persisted reservation weight = %d, want %d", persisted[0].WeightKg, weightKg)
	}
}
