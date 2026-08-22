package transportquote

import (
	"context"
	"fmt"
)

type Service struct {
	provider Provider
	breaker  *Breaker
}

func NewService(provider Provider, threshold int) *Service {
	return &Service{provider: provider, breaker: NewBreaker(threshold)}
}
func (s *Service) Get(ctx context.Context, request Request) (Quote, error) {
	if request.TenantID == "" || request.RouteID == "" || request.WeightKG < 1 {
		return Quote{}, fmt.Errorf("transport quote: invalid request")
	}
	if err := s.breaker.Allow(); err != nil {
		return Quote{}, err
	}
	quote, err := s.provider.Quote(ctx, request)
	if err != nil {
		s.breaker.Failure()
		return Quote{}, fmt.Errorf("provider quote: %w", err)
	}
	s.breaker.Success()
	if quote.RouteID != request.RouteID {
		return Quote{}, fmt.Errorf("transport quote: route mismatch")
	}
	return quote, nil
}
