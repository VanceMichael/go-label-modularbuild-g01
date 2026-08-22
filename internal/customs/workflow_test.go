package quality

import (
	"context"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"testing"
	"time"
)

func TestQualityReview(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	w := NewWorkflow()
	if err := w.Open(context.Background(), Case{ID: "C", TenantID: "T", ModuleMoveID: "S"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Attach(context.Background(), "S", Document{Number: "D1", IssuedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	c, err := w.Review(context.Background(), "S", "u", now.Add(time.Minute))
	if err != nil || c.Status != domain.QualityReleased {
		t.Fatalf("review: %v %#v", err, c)
	}
}
func TestExpiredDocumentHeld(t *testing.T) {
	now := time.Now().UTC()
	w := NewWorkflow()
	_ = w.Open(context.Background(), Case{ID: "C", TenantID: "T", ModuleMoveID: "S"})
	_ = w.Attach(context.Background(), "S", Document{Number: "D", IssuedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour)})
	c, err := w.Review(context.Background(), "S", "u", now)
	if err != domain.ErrExpired || c.Status != domain.QualityHeld {
		t.Fatalf("want held, got %v %#v", err, c)
	}
}
