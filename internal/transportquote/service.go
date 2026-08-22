package transportquote

import (
	"context"
	"fmt"
	"sync"
)

type Service struct {
	provider  Provider
	threshold int

	mu       sync.Mutex
	breakers map[string]*Breaker
}

func NewService(provider Provider, threshold int) *Service {
	return &Service{provider: provider, threshold: threshold, breakers: make(map[string]*Breaker)}
}

// breakerFor returns the breaker scoped to a single tenant so that one
// tenant's provider failures cannot trip the breaker of any other tenant.
func (s *Service) breakerFor(tenantID string) *Breaker {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.breakers[tenantID]
	if !ok {
		b = NewBreaker(s.threshold)
		s.breakers[tenantID] = b
	}
	return b
}

func (s *Service) Get(ctx context.Context, request Request) (Quote, error) {
	if request.TenantID == "" || request.RouteID == "" || request.WeightKG < 1 {
		return Quote{}, fmt.Errorf("transport quote: invalid request")
	}
	breaker := s.breakerFor(request.TenantID)
	if err := breaker.Allow(); err != nil {
		return Quote{}, err
	}
	quote, err := s.provider.Quote(ctx, request)
	if err != nil {
		breaker.Failure()
		return Quote{}, fmt.Errorf("provider quote: %w", err)
	}
	breaker.Success()
	if quote.RouteID != request.RouteID {
		return Quote{}, fmt.Errorf("transport quote: route mismatch")
	}
	return quote, nil
}
