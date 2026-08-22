package liftanalysis

import (
	"context"
	"fmt"
)

func (c *Coordinator) Run(ctx context.Context, request Request) (Result, error) {
	if request.TenantID == "" || request.PlanID == "" || request.Revision < 1 {
		return Result{}, fmt.Errorf("lift analysis: invalid request")
	}
	result, err := c.engine.Analyze(c.operationContext(ctx), request)
	if err != nil {
		return Result{}, fmt.Errorf("analyze lift plan %s: %w", request.PlanID, err)
	}
	if result.PlanID != request.PlanID {
		return Result{}, fmt.Errorf("lift analysis: result belongs to another plan")
	}
	c.reset()
	return result, nil
}
