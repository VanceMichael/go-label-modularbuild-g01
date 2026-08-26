package windpolicy

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	cache *Cache
	now   func() time.Time
}

func NewService(cache *Cache, now func() time.Time) *Service {
	return &Service{cache: cache, now: now}
}

func (s *Service) Evaluate(ctx context.Context, tenantID string, observedMPS int) (Evaluation, error) {
	if observedMPS < 0 {
		return Evaluation{}, fmt.Errorf("wind policy: observed speed cannot be negative")
	}
	policy, err := s.cache.Active(ctx, tenantID)
	if err != nil {
		return Evaluation{}, err
	}
	if policy.EffectiveFrom.After(s.now()) {
		return Evaluation{}, fmt.Errorf("wind policy: active policy is not effective yet")
	}
	return Evaluation{
		Allowed:     observedMPS <= policy.MaxWindMPS,
		ObservedMPS: observedMPS,
		LimitMPS:    policy.MaxWindMPS,
	}, nil
}
