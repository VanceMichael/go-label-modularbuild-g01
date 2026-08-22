package quality

import (
	"context"
	"errors"
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

func TestAttachBatchLaterInvalidLeavesDocumentsEmpty(t *testing.T) {
	now := time.Now().UTC()
	w := NewWorkflow()
	if err := w.Open(context.Background(), Case{ID: "case-1", TenantID: "tenant-1", ModuleMoveID: "module-1"}); err != nil {
		t.Fatal(err)
	}
	valid := Document{Number: "CERT-VALID", Kind: "lift-certificate", IssuedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
	badRange := Document{Number: "CERT-EXPIRED-RANGE", Kind: "route-permit", IssuedAt: now.Add(time.Hour), ExpiresAt: now}

	err := w.AttachBatch(context.Background(), "module-1", []Document{valid, badRange})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	c, err := w.Get("module-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Documents) != 0 {
		t.Fatalf("want empty documents after failed batch, got %#v", c.Documents)
	}
}

func TestAttachBatchValidKeepsSubmitOrder(t *testing.T) {
	now := time.Now().UTC()
	w := NewWorkflow()
	if err := w.Open(context.Background(), Case{ID: "case-2", TenantID: "tenant-1", ModuleMoveID: "module-2"}); err != nil {
		t.Fatal(err)
	}
	cert1 := Document{Number: "CERT-1", Kind: "lift-certificate", IssuedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
	cert2 := Document{Number: "CERT-2", Kind: "route-permit", IssuedAt: now, ExpiresAt: now.Add(24 * time.Hour)}

	if err := w.AttachBatch(context.Background(), "module-2", []Document{cert1, cert2}); err != nil {
		t.Fatalf("attach batch: %v", err)
	}
	c, err := w.Get("module-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Documents) != 2 || c.Documents[0].Number != "CERT-1" || c.Documents[1].Number != "CERT-2" {
		t.Fatalf("want CERT-1,CERT-2 in order, got %#v", c.Documents)
	}
}
