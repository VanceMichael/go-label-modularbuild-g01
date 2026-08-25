package booking

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type cancellationBetweenChecks struct {
	context.Context
	calls  atomic.Int32
	open   chan struct{}
	closed chan struct{}
}

func newCancellationBetweenChecks() *cancellationBetweenChecks {
	closed := make(chan struct{})
	close(closed)
	return &cancellationBetweenChecks{
		Context: context.Background(),
		open:    make(chan struct{}),
		closed:  closed,
	}
}

func (c *cancellationBetweenChecks) Done() <-chan struct{} {
	if c.calls.Add(1) == 1 {
		return c.open
	}
	return c.closed
}

func (c *cancellationBetweenChecks) Err() error {
	if c.calls.Load() >= 2 {
		return context.Canceled
	}
	return nil
}

func TestCancellationDuringReservationDoesNotConsumeCapacity(t *testing.T) {
	t.Run("mid-operation cancellation leaves no hold", func(t *testing.T) {
		ledger := NewLedger()
		if err := ledger.DefineLeg("lift-17", 20); err != nil {
			t.Fatal(err)
		}
		request := Request{TenantID: "tenant-1", ModuleMoveID: "module-17", LegID: "lift-17", WeightKg: 7}
		if _, err := ledger.Reserve(newCancellationBetweenChecks(), request); !errors.Is(err, context.Canceled) {
			t.Fatalf("reservation error = %v, want cancellation", err)
		}
		available, err := ledger.Available("lift-17")
		if err != nil || available != 20 {
			t.Errorf("capacity after cancellation = %d, err=%v; want 20", available, err)
		}
		if allocations := ledger.List("tenant-1"); len(allocations) != 0 {
			t.Errorf("cancelled reservation created allocations: %+v", allocations)
		}
	})

	t.Run("active request reserves and releases normally", func(t *testing.T) {
		ledger := NewLedger()
		if err := ledger.DefineLeg("lift-18", 20); err != nil {
			t.Fatal(err)
		}
		allocation, err := ledger.Reserve(context.Background(), Request{TenantID: "tenant-1", ModuleMoveID: "module-18", LegID: "lift-18", WeightKg: 7})
		if err != nil || !allocation.Accepted {
			t.Fatalf("legal reservation: allocation=%+v err=%v", allocation, err)
		}
		available, _ := ledger.Available("lift-18")
		if available != 13 || len(ledger.List("tenant-1")) != 1 {
			t.Fatalf("legal reservation state: available=%d allocations=%d", available, len(ledger.List("tenant-1")))
		}
		if err := ledger.Release(allocation.ID); err != nil {
			t.Fatal(err)
		}
		available, _ = ledger.Available("lift-18")
		if available != 20 {
			t.Fatalf("released capacity = %d, want 20", available)
		}
	})
}
