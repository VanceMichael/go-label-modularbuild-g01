package liftanalysis_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/liftanalysis"
)

type controlledEngine struct {
	started   chan string
	probeB    chan struct{}
	observedB chan error
	releaseB  chan struct{}
}

func (e *controlledEngine) Analyze(ctx context.Context, request liftanalysis.Request) (liftanalysis.Result, error) {
	e.started <- request.PlanID
	switch request.PlanID {
	case "plan-a":
		<-ctx.Done()
		return liftanalysis.Result{}, ctx.Err()
	case "plan-b":
		<-e.probeB
		e.observedB <- ctx.Err()
		<-e.releaseB
		if err := ctx.Err(); err != nil {
			return liftanalysis.Result{}, err
		}
	}
	return liftanalysis.Result{PlanID: request.PlanID, Safe: true}, nil
}

func TestCancelingOneAnalysisDoesNotAbortAnotherPlan(t *testing.T) {
	engine := &controlledEngine{
		started: make(chan string, 3), probeB: make(chan struct{}),
		observedB: make(chan error, 1), releaseB: make(chan struct{}),
	}
	coordinator := liftanalysis.NewCoordinator(engine)
	ctxA, cancelA := context.WithCancel(context.Background())
	type outcome struct {
		result liftanalysis.Result
		err    error
	}
	results := make(chan outcome, 2)
	go func() {
		result, err := coordinator.Run(ctxA, liftanalysis.Request{TenantID: "tenant-site-a", PlanID: "plan-a", Revision: 2})
		results <- outcome{result: result, err: err}
	}()
	if plan := <-engine.started; plan != "plan-a" {
		t.Fatalf("first started plan = %s", plan)
	}
	go func() {
		result, err := coordinator.Run(context.Background(), liftanalysis.Request{TenantID: "tenant-site-a", PlanID: "plan-b", Revision: 4})
		results <- outcome{result: result, err: err}
	}()
	if plan := <-engine.started; plan != "plan-b" {
		t.Fatalf("second started plan = %s", plan)
	}

	cancelA()
	close(engine.probeB)
	if err := <-engine.observedB; err != nil {
		t.Errorf("plan-b context after canceling plan-a = %v, want active", err)
	}
	close(engine.releaseB)

	var canceled, succeeded int
	for range 2 {
		out := <-results
		if errors.Is(out.err, context.Canceled) {
			canceled++
			continue
		}
		if out.err == nil && out.result.PlanID == "plan-b" && out.result.Safe {
			succeeded++
			continue
		}
		t.Errorf("unexpected concurrent outcome: %#v", out)
	}
	if canceled != 1 || succeeded != 1 {
		t.Errorf("outcome counts canceled=%d succeeded=%d, want 1/1", canceled, succeeded)
	}

	result, err := coordinator.Run(context.Background(), liftanalysis.Request{TenantID: "tenant-site-a", PlanID: "plan-c", Revision: 1})
	if err != nil || result.PlanID != "plan-c" || !result.Safe {
		t.Fatalf("independent plan-c result = %#v, %v", result, err)
	}
	if plan := <-engine.started; plan != "plan-c" {
		t.Fatalf("third started plan = %s", plan)
	}
}
