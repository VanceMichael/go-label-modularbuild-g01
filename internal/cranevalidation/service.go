package cranevalidation

import (
	"context"
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
	result, err := s.validator.Validate(ctx, r)
	if err != nil {
		return Result{}, fmt.Errorf("validate crane: %w", err)
	}
	s.leases.Release(key)
	if result.CraneID != r.CraneID {
		return Result{}, fmt.Errorf("crane validation: result mismatch")
	}
	return result, nil
}
