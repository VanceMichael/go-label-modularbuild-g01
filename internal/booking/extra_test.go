package booking

import (
	"context"
	"errors"
	"testing"
)

// TestReserveCancellationRollback reproduces the on-site failure where a
// cancellation arrives after the entry check but before the allocation is
// saved. Reserve must return context.Canceled, leave the leg fully available,
// and keep the tenant allocation list empty. Field labels: lift-17 / tenant-1 / module-17, 20kg window.
func TestReserveCancellationRollback(t *testing.T) {
	l := NewLedger()
	if err := l.DefineLeg("lift-17", 20); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Cancel the context in the window between the capacity deduction and the
	// in-flight cancellation check, deterministically reproducing a
	// cancellation that lands after the entry check but before the save.
	hookAfterReserve = func() { cancel() }
	t.Cleanup(func() { hookAfterReserve = nil })

	_, err := l.Reserve(ctx, Request{TenantID: "tenant-1", ModuleMoveID: "module-17", LegID: "lift-17", WeightKg: 7})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}

	// Failure cleanup: window fully available, tenant list empty.
	if available, _ := l.Available("lift-17"); available != 20 {
		t.Fatalf("leg leaked reserved capacity: available=%d want 20", available)
	}
	if got := len(l.List("tenant-1")); got != 0 {
		t.Fatalf("tenant list not empty: %d", got)
	}
}

// TestReserveRetryAfterCancellation covers the recovery-retry path: after the
// rolled-back cancellation, a subsequent reserve on the same leg still succeeds.
func TestReserveRetryAfterCancellation(t *testing.T) {
	l := NewLedger()
	_ = l.DefineLeg("lift-18", 20)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := l.Reserve(ctx, Request{TenantID: "tenant-1", ModuleMoveID: "module-18", LegID: "lift-18", WeightKg: 7}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}

	// Recovery retry on the same leg succeeds; capacity was not leaked.
	a, err := l.Reserve(context.Background(), Request{TenantID: "tenant-1", ModuleMoveID: "module-18", LegID: "lift-18", WeightKg: 7})
	if err != nil || !a.Accepted {
		t.Fatalf("retry reserve: %v %+v", err, a)
	}
	if available, _ := l.Available("lift-18"); available != 13 {
		t.Fatalf("available=%d want 13", available)
	}
}

// TestReserveNormalFlow covers the normal handling path: a 7kg reservation
// yields an accepted allocation with 13kg remaining, and Release restores 20kg.
func TestReserveNormalFlow(t *testing.T) {
	l := NewLedger()
	if err := l.DefineLeg("lift-17", 20); err != nil {
		t.Fatal(err)
	}
	a, err := l.Reserve(context.Background(), Request{TenantID: "tenant-1", ModuleMoveID: "module-17", LegID: "lift-17", WeightKg: 7})
	if err != nil || !a.Accepted {
		t.Fatalf("reserve: %v %+v", err, a)
	}
	if available, _ := l.Available("lift-17"); available != 13 {
		t.Fatalf("available=%d want 13", available)
	}
	if err := l.Release(a.ID); err != nil {
		t.Fatal(err)
	}
	if available, _ := l.Available("lift-17"); available != 20 {
		t.Fatalf("release did not restore: available=%d want 20", available)
	}
}
