package windpolicy

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// stubLoader records calls and lets each call's outcome be controlled.
type stubLoader struct {
	mu      sync.Mutex
	calls   []string
	results []func(context.Context, string) (Policy, error)
}

func (s *stubLoader) LoadActive(ctx context.Context, tenantID string) (Policy, error) {
	s.mu.Lock()
	idx := len(s.calls)
	s.calls = append(s.calls, tenantID)
	results := s.results
	s.mu.Unlock()

	if idx < len(results) {
		return results[idx](ctx, tenantID)
	}
	return Policy{TenantID: tenantID, MaxWindMPS: 12}, nil
}

func (s *stubLoader) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// loaderStartedThenCancelled simulates the reported scenario: the loader has
// started (ctx is already non-cancelled when entered) but the caller cancels
// mid-load. The closure blocks until it observes cancellation.
func loaderStartedThenCancelled(ctx context.Context, tenantID string) (Policy, error) {
	// Prove the loader actually started before cancellation propagated.
	select {
	case <-ctx.Done():
		return Policy{}, ctx.Err()
	case <-time.After(50 * time.Millisecond):
	}
	// Now cancel on the caller's side by waiting on the parent; the test
	// cancels ctx from another goroutine, so this returns ctx.Err().
	<-ctx.Done()
	return Policy{}, ctx.Err()
}

func TestActiveCachesSuccessfulLoad(t *testing.T) {
	l := &stubLoader{}
	c := NewCache(l)

	p1, err := c.Active(context.Background(), "tenant-site-a")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if p1.TenantID != "tenant-site-a" {
		t.Fatalf("tenant mismatch: %+v", p1)
	}
	p2, err := c.Active(context.Background(), "tenant-site-a")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if p2 != p1 {
		t.Fatalf("policy not cached: got %+v want %+v", p2, p1)
	}
	if got := l.callCount(); got != 1 {
		t.Fatalf("loader invoked %d times, want 1", got)
	}
}

func TestActiveCancellationNotCached(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	l := &stubLoader{
		results: []func(context.Context, string) (Policy, error){
			loaderStartedThenCancelled,
			func(ctx context.Context, tenantID string) (Policy, error) {
				return Policy{TenantID: tenantID, MaxWindMPS: 9}, nil
			},
		},
	}
	c := NewCache(l)

	// Cancel after the loader has started.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := c.Active(ctx, "tenant-site-a")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}

	// The failed (cancelled) load must not be cached: a fresh context must
	// retry the loader and succeed.
	p, err := c.Active(context.Background(), "tenant-site-a")
	if err != nil {
		t.Fatalf("retry after cancellation: %v", err)
	}
	if p.MaxWindMPS != 9 {
		t.Fatalf("retry did not load fresh policy: %+v", p)
	}
	if got := l.callCount(); got != 2 {
		t.Fatalf("loader invoked %d times after retry, want 2", got)
	}
}

func TestActivePreCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	l := &stubLoader{}
	c := NewCache(l)

	_, err := c.Active(ctx, "tenant-site-b")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	// Pre-check must short-circuit before the loader runs.
	if got := l.callCount(); got != 0 {
		t.Fatalf("loader invoked %d times, want 0", got)
	}

	// And the cancellation is not cached: a fresh context retries.
	if _, err := c.Active(context.Background(), "tenant-site-b"); err != nil {
		t.Fatalf("retry after pre-cancellation: %v", err)
	}
	if got := l.callCount(); got != 1 {
		t.Fatalf("loader invoked %d times after retry, want 1", got)
	}
}

func TestActiveCachesOtherErrors(t *testing.T) {
	boom := errors.New("boom")
	l := &stubLoader{
		results: []func(context.Context, string) (Policy, error){
			func(context.Context, string) (Policy, error) { return Policy{}, boom },
		},
	}
	c := NewCache(l)

	_, err := c.Active(context.Background(), "tenant-site-b")
	if !errors.Is(err, boom) {
		t.Fatalf("want boom, got %v", err)
	}
	// Non-cancellation errors are cached.
	_, err = c.Active(context.Background(), "tenant-site-b")
	if !errors.Is(err, boom) {
		t.Fatalf("want cached boom, got %v", err)
	}
	if got := l.callCount(); got != 1 {
		t.Fatalf("loader invoked %d times, want 1", got)
	}
}

func TestActiveRejectsCrossTenant(t *testing.T) {
	l := &stubLoader{
		results: []func(context.Context, string) (Policy, error){
			func(context.Context, string) (Policy, error) {
				return Policy{TenantID: "other"}, nil
			},
		},
	}
	c := NewCache(l)

	_, err := c.Active(context.Background(), "tenant-site-a")
	if err == nil {
		t.Fatal("want cross-tenant error, got nil")
	}
	// The cross-tenant failure is cached.
	if _, err = c.Active(context.Background(), "tenant-site-a"); err == nil {
		t.Fatal("want cached cross-tenant error, got nil")
	}
	if got := l.callCount(); got != 1 {
		t.Fatalf("loader invoked %d times, want 1", got)
	}
}
