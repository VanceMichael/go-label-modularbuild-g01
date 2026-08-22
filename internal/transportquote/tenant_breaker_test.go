package transportquote_test

import (
	"context"
	"errors"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/transportquote"
	"testing"
)

var errProvider = errors.New("provider unavailable")

type provider struct{ calls map[string]int }

func (p *provider) Quote(_ context.Context, r transportquote.Request) (transportquote.Quote, error) {
	p.calls[r.TenantID]++
	if r.TenantID == "tenant-a" {
		return transportquote.Quote{}, errProvider
	}
	return transportquote.Quote{RouteID: r.RouteID, AmountCents: 42000}, nil
}
func TestCircuitStateIsIsolatedPerTenant(t *testing.T) {
	p := &provider{calls: map[string]int{}}
	s := transportquote.NewService(p, 2)
	for range 2 {
		if _, err := s.Get(context.Background(), transportquote.Request{TenantID: "tenant-a", RouteID: "route-a", WeightKG: 7000}); !errors.Is(err, errProvider) {
			t.Errorf("tenant-a failure=%v", err)
		}
	}
	q, err := s.Get(context.Background(), transportquote.Request{TenantID: "tenant-b", RouteID: "route-b", WeightKG: 5000})
	if err != nil {
		t.Errorf("tenant-b quote error=%v", err)
	}
	if q.RouteID != "route-b" || q.AmountCents != 42000 {
		t.Errorf("tenant-b quote=%#v", q)
	}
	if p.calls["tenant-b"] != 1 {
		t.Errorf("tenant-b provider calls=%d", p.calls["tenant-b"])
	}
	if _, err := s.Get(context.Background(), transportquote.Request{TenantID: "tenant-a", RouteID: "route-a", WeightKG: 7000}); !errors.Is(err, transportquote.ErrCircuitOpen) {
		t.Errorf("tenant-a after threshold=%v", err)
	}
	clean := transportquote.NewService(p, 2)
	q, err = clean.Get(context.Background(), transportquote.Request{TenantID: "tenant-c", RouteID: "route-c", WeightKG: 3000})
	if err != nil || q.RouteID != "route-c" {
		t.Fatalf("clean quote=%#v,%v", q, err)
	}
}
