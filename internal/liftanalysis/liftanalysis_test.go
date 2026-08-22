package liftanalysis

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// probeEngine simulates a controlled probe point inside the engine. Each plan
// that enters Analyze parks on its probe until the test releases it, and it
// records the context it was handed so the test can inspect the context's
// state at the probe while a plan is still in flight.
type probeEngine struct {
	mu      sync.Mutex
	probes  map[string]chan struct{}
	ready   map[string]chan struct{}
	results map[string]Result
	ctxs    map[string]context.Context
}

func newProbeEngine() *probeEngine {
	return &probeEngine{
		probes:  make(map[string]chan struct{}),
		ready:  make(map[string]chan struct{}),
		results: make(map[string]Result),
		ctxs:    make(map[string]context.Context),
	}
}

func (e *probeEngine) register(planID string, result Result) {
	e.mu.Lock()
	e.probes[planID] = make(chan struct{})
	e.ready[planID] = make(chan struct{})
	e.results[planID] = result
	e.mu.Unlock()
}

func (e *probeEngine) release(planID string) {
	e.mu.Lock()
	if ch, ok := e.probes[planID]; ok {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
	e.mu.Unlock()
}

func (e *probeEngine) ctxFor(planID string) context.Context {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.ctxs[planID]
}

func (e *probeEngine) Analyze(ctx context.Context, req Request) (Result, error) {
	e.mu.Lock()
	ch := e.probes[req.PlanID]
	ready := e.ready[req.PlanID]
	result := e.results[req.PlanID]
	e.ctxs[req.PlanID] = ctx
	e.mu.Unlock()
	if ch == nil {
		return Result{}, errors.New("probe not registered for " + req.PlanID)
	}
	// Signal that this plan has reached its controlled probe point.
	close(ready)
	// Controlled probe point: block until released or until THIS plan's own
	// context is cancelled. A sibling plan being cancelled must NOT land here.
	select {
	case <-ch:
		return result, nil
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

// TestCoordinatorCancelIsolatesPlans is the on-site flow for business labels
// plan-a, plan-b, tenant-site-a and plan-c:
//   - plan-a and plan-b both enter the engine.
//   - plan-a is cancelled; it must return context.Canceled.
//   - plan-b's context at the controlled probe point must still be active, and
//     after being released it returns its own safe result.
//   - cancelling a sibling that is not in flight (plan-c) is a harmless no-op.
func TestCoordinatorCancelIsolatesPlans(t *testing.T) {
	engine := newProbeEngine()
	engine.register("plan-a", Result{})
	engine.register("plan-b", Result{PlanID: "plan-b", Safe: true})
	coordinator := NewCoordinator(engine)

	var (
		aErr, bErr       error
		aResult, bResult Result
	)
	doneA := make(chan struct{})
	doneB := make(chan struct{})

	// plan-a enters the engine.
	go func() {
		aResult, aErr = coordinator.Run(context.Background(), Request{
			TenantID: "tenant-site-a", PlanID: "plan-a", Revision: 1,
		})
		close(doneA)
	}()

	// plan-b enters the engine.
	go func() {
		bResult, bErr = coordinator.Run(context.Background(), Request{
			TenantID: "tenant-site-a", PlanID: "plan-b", Revision: 1,
		})
		close(doneB)
	}()

	// Wait until both plans are parked at their controlled probe points.
	<-engine.ready["plan-a"]
	<-engine.ready["plan-b"]

	// Cancel plan-a while plan-b is still in flight at its probe. Cancelling
	// plan-c (not in the engine) must be a harmless no-op here too.
	coordinator.Cancel("plan-a")
	coordinator.Cancel("plan-c")

	// plan-a must surface context.Canceled, unwrapped.
	<-doneA
	if !errors.Is(aErr, context.Canceled) {
		t.Fatalf("plan-a: want context.Canceled, got %v", aErr)
	}
	if aResult != (Result{}) {
		t.Fatalf("plan-a: want zero result, got %+v", aResult)
	}

	// plan-b must still be in flight: its context at the probe must be active,
	// and it must not have returned yet.
	if bCtx := engine.ctxFor("plan-b"); bCtx == nil || bCtx.Err() != nil {
		t.Fatalf("plan-b context not active at probe: %v", bCtx.Err())
	}
	select {
	case <-doneB:
		t.Fatal("plan-b returned before being released")
	default:
	}

	// Release plan-b through its probe; it must return its own safe result.
	engine.release("plan-b")
	<-doneB
	if bErr != nil {
		t.Fatalf("plan-b: want nil error, got %v", bErr)
	}
	if bResult.PlanID != "plan-b" {
		t.Fatalf("plan-b: want result belongs to plan-b, got %+v", bResult)
	}
	if !bResult.Safe {
		t.Fatalf("plan-b: want safe result, got %+v", bResult)
	}
}
