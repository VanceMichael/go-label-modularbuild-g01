package transportquote

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

// stubProvider counts calls and returns ErrProviderUnavailable until the
// configured call, then returns a fixed quote.
type stubProvider struct {
	calls       atomic.Int64
	unavailable int64 // number of leading calls that report unavailable
	quote       Quote
}

var errProviderUnavailable = errors.New("provider unavailable")

func (p *stubProvider) Quote(ctx context.Context, req Request) (Quote, error) {
	n := p.calls.Add(1)
	if n <= p.unavailable {
		return Quote{}, errProviderUnavailable
	}
	q := p.quote
	q.RouteID = req.RouteID
	return q, nil
}

// TestTenantIsolatedBreaker reproduces the on-site flow:
//   - tenant-a fails twice ("provider unavailable") and opens only its own breaker
//   - tenant-b's route-b actually calls the provider once and returns 42000 cents
//   - tenant-a's next request is still ErrCircuitOpen
func TestTenantIsolatedBreaker(t *testing.T) {
	const threshold = 2
	provider := &stubProvider{unavailable: 2, quote: Quote{AmountCents: 42000}}
	svc := NewService(provider, threshold)

	// tenant-a: two provider failures open tenant-a's breaker only.
	for i := 0; i < threshold; i++ {
		if _, err := svc.Get(context.Background(), Request{TenantID: "tenant-a", RouteID: "route-a", WeightKG: 1}); err == nil {
			t.Fatalf("tenant-a call %d: expected provider failure", i+1)
		}
	}

	// tenant-b / route-b must actually reach the provider and return 42000 cents.
	q, err := svc.Get(context.Background(), Request{TenantID: "tenant-b", RouteID: "route-b", WeightKG: 1})
	if err != nil {
		t.Fatalf("tenant-b: expected success, got %v", err)
	}
	if q.AmountCents != 42000 {
		t.Fatalf("tenant-b: expected 42000 cents, got %d", q.AmountCents)
	}
	if got := provider.calls.Load(); got != 3 {
		t.Fatalf("provider calls: expected 3 (2 tenant-a + 1 tenant-b), got %d", got)
	}

	// tenant-a's next request is still ErrCircuitOpen (its own breaker is open).
	_, err = svc.Get(context.Background(), Request{TenantID: "tenant-a", RouteID: "route-a", WeightKG: 1})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("tenant-a: expected ErrCircuitOpen, got %v", err)
	}

	// tenant-a's open breaker must not block tenant-c / route-c.
	if _, err := svc.Get(context.Background(), Request{TenantID: "tenant-c", RouteID: "route-c", WeightKG: 1}); err != nil {
		t.Fatalf("tenant-c: expected success, got %v", err)
	}
}
