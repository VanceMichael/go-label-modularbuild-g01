package crane

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type synchronizedReads struct {
	Store
	reads   atomic.Int32
	ready   chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *synchronizedReads) GetCrane(ctx context.Context, tenantID, craneID string) (Crane, error) {
	crane, err := s.Store.GetCrane(ctx, tenantID, craneID)
	if err != nil {
		return Crane{}, err
	}
	if s.reads.Add(1) == 2 {
		s.once.Do(func() { close(s.ready) })
	}
	select {
	case <-s.release:
		return crane, nil
	case <-ctx.Done():
		return Crane{}, ctx.Err()
	}
}

func TestConcurrentCraneReservationsNeverExceedCapacity(t *testing.T) {
	t.Run("competing reservations admit only one module", func(t *testing.T) {
		memory := NewMemory()
		if err := memory.AddCrane(Crane{ID: "crane-17", TenantID: "tenant-1", CapacityKg: 10}); err != nil {
			t.Fatal(err)
		}
		barrier := &synchronizedReads{Store: memory, ready: make(chan struct{}), release: make(chan struct{})}
		scheduler := NewScheduler(barrier)
		requests := []Reservation{
			{ID: "reservation-a", TenantID: "tenant-1", CraneID: "crane-17", ModuleMoveID: "module-a", WeightKg: 7},
			{ID: "reservation-b", TenantID: "tenant-1", CraneID: "crane-17", ModuleMoveID: "module-b", WeightKg: 7},
		}
		errorsByRequest := make(chan error, len(requests))
		for _, request := range requests {
			request := request
			go func() { errorsByRequest <- scheduler.Reserve(context.Background(), request) }()
		}
		<-barrier.ready
		close(barrier.release)

		succeeded, capacityRejected := 0, 0
		for range requests {
			err := <-errorsByRequest
			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, domain.ErrCapacity):
				capacityRejected++
			default:
				t.Errorf("unexpected reservation error: %v", err)
			}
		}
		crane, err := memory.GetCrane(context.Background(), "tenant-1", "crane-17")
		reservations, listErr := memory.Reservations(context.Background(), "tenant-1", "crane-17")
		if err != nil || listErr != nil || succeeded != 1 || capacityRejected != 1 || crane.ReservedKg != 7 || len(reservations) != 1 {
			t.Errorf("concurrent capacity result: succeeded=%d rejected=%d crane=%+v reservations=%d errors=(%v,%v)", succeeded, capacityRejected, crane, len(reservations), err, listErr)
		}
	})

	t.Run("sequential reservations can fill the crane exactly", func(t *testing.T) {
		memory := NewMemory()
		if err := memory.AddCrane(Crane{ID: "crane-legal", TenantID: "tenant-1", CapacityKg: 10}); err != nil {
			t.Fatal(err)
		}
		scheduler := NewScheduler(memory)
		for _, request := range []Reservation{
			{ID: "reservation-4", TenantID: "tenant-1", CraneID: "crane-legal", ModuleMoveID: "module-4", WeightKg: 4},
			{ID: "reservation-6", TenantID: "tenant-1", CraneID: "crane-legal", ModuleMoveID: "module-6", WeightKg: 6},
		} {
			if err := scheduler.Reserve(context.Background(), request); err != nil {
				t.Fatalf("legal reservation %s: %v", request.ID, err)
			}
		}
		crane, err := memory.GetCrane(context.Background(), "tenant-1", "crane-legal")
		reservations, listErr := memory.Reservations(context.Background(), "tenant-1", "crane-legal")
		if err != nil || listErr != nil || crane.ReservedKg != 10 || len(reservations) != 2 {
			t.Fatalf("legal capacity result: crane=%+v reservations=%d errors=(%v,%v)", crane, len(reservations), err, listErr)
		}
	})
}
