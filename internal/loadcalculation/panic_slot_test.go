package loadcalculation_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/loadcalculation"
)

type calculationHandler struct{ panicPlan string }

func (h calculationHandler) Calculate(_ context.Context, request loadcalculation.Request) (loadcalculation.Result, error) {
	if request.PlanID == h.panicPlan {
		panic("solver matrix corrupted")
	}
	return loadcalculation.Result{PlanID: request.PlanID, PeakKG: 6800}, nil
}

func TestRecoveredCalculationPanicReleasesCapacity(t *testing.T) {
	limiter := loadcalculation.NewLimiter(1)
	runner := loadcalculation.NewRunner(limiter, calculationHandler{panicPlan: "plan-panic"})
	_, err := runner.Run(context.Background(), loadcalculation.Request{PlanID: "plan-panic", Revision: 2})
	if err == nil || !strings.Contains(err.Error(), "solver matrix corrupted") {
		t.Fatalf("panic Run() error = %v", err)
	}
	if got := limiter.InUse(); got != 0 {
		t.Errorf("capacity in use after recovered panic = %d, want 0", got)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result, err := runner.Run(ctx, loadcalculation.Request{PlanID: "plan-next", Revision: 3})
	if err != nil {
		t.Errorf("next Run() error = %v", err)
	}
	if result.PlanID != "plan-next" || result.PeakKG != 6800 {
		t.Errorf("next result = %#v", result)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Error("next calculation exhausted deadline waiting for leaked slot")
	}

	cleanLimiter := loadcalculation.NewLimiter(1)
	cleanRunner := loadcalculation.NewRunner(cleanLimiter, calculationHandler{})
	clean, err := cleanRunner.Run(context.Background(), loadcalculation.Request{PlanID: "plan-clean", Revision: 1})
	if err != nil || clean.PlanID != "plan-clean" || cleanLimiter.InUse() != 0 {
		t.Fatalf("clean run = %#v, %v, inUse=%d", clean, err, cleanLimiter.InUse())
	}
}
