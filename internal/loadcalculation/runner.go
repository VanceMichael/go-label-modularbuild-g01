package loadcalculation

import (
	"context"
	"fmt"
)

type Runner struct {
	limiter *Limiter
	handler Handler
}

func NewRunner(limiter *Limiter, handler Handler) *Runner {
	return &Runner{limiter: limiter, handler: handler}
}

func (r *Runner) Run(ctx context.Context, request Request) (result Result, err error) {
	if request.PlanID == "" || request.Revision < 1 {
		return Result{}, fmt.Errorf("load calculation: invalid request")
	}
	if err := r.limiter.Acquire(ctx); err != nil {
		return Result{}, fmt.Errorf("acquire calculation slot: %w", err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("load calculation panic: %v", recovered)
			return
		}
		r.limiter.Release()
	}()
	result, err = r.handler.Calculate(ctx, request)
	if err != nil {
		return Result{}, fmt.Errorf("calculate plan: %w", err)
	}
	return result, nil
}
