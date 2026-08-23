package cranevalidation

import (
	"context"
	"errors"
	"fmt"
)

type Service struct {
	leases    *Leases
	validator Validator
}

func NewService(l *Leases, v Validator) *Service { return &Service{leases: l, validator: v} }
func (s *Service) Run(ctx context.Context, r Request) (Result, error) {
	if r.TenantID == "" || r.CraneID == "" || r.ConfigVersion == "" {
		return Result{}, fmt.Errorf("crane validation: invalid request")
	}
	key := r.TenantID + "/" + r.CraneID
	if err := s.leases.Acquire(key); err != nil {
		return Result{}, err
	}
	// Release on every exit path so a canceled or failed validation never
	// leaves the lease held, which would block retries for the same crane.
	defer s.leases.Release(key)

	result, err := s.validator.Validate(ctx, r)
	if err != nil {
		// A canceled request (e.g. crane-12 canceled after the validator
		// started) must surface context.Canceled to the caller unwrapped.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Result{}, err
		}
		return Result{}, fmt.Errorf("validate crane: %w", err)
	}
	if result.CraneID != r.CraneID {
		return Result{}, fmt.Errorf("crane validation: result mismatch")
	}
	return result, nil
}
