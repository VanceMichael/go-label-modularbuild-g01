package liftanalysis

import (
	"context"
	"errors"
	"fmt"
)

func (c *Coordinator) Run(ctx context.Context, request Request) (Result, error) {
	if request.TenantID == "" || request.PlanID == "" || request.Revision < 1 {
		return Result{}, fmt.Errorf("lift analysis: invalid request")
	}
	scoped, cancel := c.operationContext(ctx, request.PlanID)
	defer c.release(request.PlanID, cancel)
	result, err := c.engine.Analyze(scoped, request)
	if err != nil {
		// A cancellation of this plan's own context must surface unwrapped
		// (context.Canceled) rather than as a wrapped analyze error, so that
		// callers can distinguish cancellation from a genuine engine fault.
		if errors.Is(err, context.Canceled) {
			return Result{}, context.Canceled
		}
		return Result{}, fmt.Errorf("analyze lift plan %s: %w", request.PlanID, err)
	}
	if result.PlanID != request.PlanID {
		return Result{}, fmt.Errorf("lift analysis: result belongs to another plan")
	}
	return result, nil
}
