package windpolicy_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/windpolicy"
)

type policyLoader struct {
	policy     windpolicy.Policy
	blockFirst bool
	started    chan struct{}
	calls      atomic.Int32
}

func (l *policyLoader) LoadActive(ctx context.Context, _ string) (windpolicy.Policy, error) {
	call := l.calls.Add(1)
	if l.blockFirst && call == 1 {
		close(l.started)
		<-ctx.Done()
		return windpolicy.Policy{}, ctx.Err()
	}
	return l.policy, nil
}

func TestCancelledPolicyLoadCanRecoverWithoutRestart(t *testing.T) {
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	loader := &policyLoader{
		policy:     windpolicy.Policy{TenantID: "tenant-site-a", MaxWindMPS: 15, EffectiveFrom: now.Add(-time.Hour)},
		blockFirst: true,
		started:    make(chan struct{}),
	}
	service := windpolicy.NewService(windpolicy.NewCache(loader), func() time.Time { return now })
	ctx, cancel := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() {
		_, err := service.Evaluate(ctx, "tenant-site-a", 12)
		firstResult <- err
	}()
	<-loader.started
	cancel()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Errorf("first Evaluate() error = %v, want context cancellation", err)
	}

	decision, err := service.Evaluate(context.Background(), "tenant-site-a", 12)
	if err != nil {
		t.Errorf("healthy Evaluate() after cancellation error = %v", err)
	}
	if !decision.Allowed || decision.ObservedMPS != 12 || decision.LimitMPS != 15 {
		t.Errorf("healthy decision = %#v, want allowed 12m/s under 15m/s limit", decision)
	}
	if calls := loader.calls.Load(); calls != 2 {
		t.Errorf("loader calls after recovery = %d, want canceled attempt plus retry", calls)
	}

	stableLoader := &policyLoader{
		policy:  windpolicy.Policy{TenantID: "tenant-site-b", MaxWindMPS: 10, EffectiveFrom: now.Add(-time.Hour)},
		started: make(chan struct{}),
	}
	stableService := windpolicy.NewService(windpolicy.NewCache(stableLoader), func() time.Time { return now })
	for _, observed := range []int{8, 9} {
		stableDecision, stableErr := stableService.Evaluate(context.Background(), "tenant-site-b", observed)
		if stableErr != nil || !stableDecision.Allowed || stableDecision.LimitMPS != 10 {
			t.Fatalf("stable Evaluate(%d) = %#v, %v", observed, stableDecision, stableErr)
		}
	}
	if calls := stableLoader.calls.Load(); calls != 1 {
		t.Fatalf("successful policy was loaded %d times, want one cached load", calls)
	}
}
