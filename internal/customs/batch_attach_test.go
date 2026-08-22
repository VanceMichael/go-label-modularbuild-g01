package quality

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

func TestBatchAttachmentIsAtomicWhenOneDocumentIsInvalid(t *testing.T) {
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)

	t.Run("invalid batch leaves case unchanged", func(t *testing.T) {
		workflow := NewWorkflow()
		if err := workflow.Open(context.Background(), Case{ID: "case-1", ModuleMoveID: "module-1", TenantID: "tenant-1"}); err != nil {
			t.Fatal(err)
		}
		documents := []Document{
			{Number: "CERT-VALID", Kind: "lift-certificate", IssuedAt: now, ExpiresAt: now.Add(24 * time.Hour)},
			{Number: "CERT-EXPIRED-RANGE", Kind: "route-permit", IssuedAt: now, ExpiresAt: now.Add(-time.Hour)},
		}
		if err := workflow.AttachBatch(context.Background(), "module-1", documents); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("invalid batch error = %v, want invalid", err)
		}
		stored, err := workflow.Get("module-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(stored.Documents) != 0 {
			t.Errorf("invalid batch persisted partial documents: %+v", stored.Documents)
		}
	})

	t.Run("valid batch persists every document", func(t *testing.T) {
		workflow := NewWorkflow()
		if err := workflow.Open(context.Background(), Case{ID: "case-2", ModuleMoveID: "module-2", TenantID: "tenant-1"}); err != nil {
			t.Fatal(err)
		}
		documents := []Document{
			{Number: "CERT-1", Kind: "lift-certificate", IssuedAt: now, ExpiresAt: now.Add(24 * time.Hour)},
			{Number: "CERT-2", Kind: "route-permit", IssuedAt: now, ExpiresAt: now.Add(48 * time.Hour)},
		}
		if err := workflow.AttachBatch(context.Background(), "module-2", documents); err != nil {
			t.Fatal(err)
		}
		stored, err := workflow.Get("module-2")
		if err != nil || len(stored.Documents) != 2 || stored.Documents[0].Number != "CERT-1" || stored.Documents[1].Number != "CERT-2" {
			t.Fatalf("valid batch state: case=%+v err=%v", stored, err)
		}
	})
}
