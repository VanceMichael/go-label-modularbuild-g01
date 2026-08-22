package handover_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/handover"
)

type certificationChecker struct {
	rejectedID string
}

func (c certificationChecker) Check(_ context.Context, module handover.Module) error {
	if module.ID == c.rejectedID {
		return handover.ErrCertificationRejected
	}
	return nil
}

func TestBatchHandoverKeepsCertificationFailureAtomic(t *testing.T) {
	fixed := time.Date(2026, 8, 22, 9, 30, 0, 0, time.UTC)
	store := handover.NewMemoryStore()
	store.Seed(
		handover.Batch{ID: "batch-42", TenantID: "tenant-site-a", ModuleIDs: []string{"module-a", "module-b"}, Status: handover.BatchPending},
		handover.Module{ID: "module-a", BatchID: "batch-42", Status: handover.ModuleStaged, Revision: 4},
		handover.Module{ID: "module-b", BatchID: "batch-42", Status: handover.ModuleStaged, Revision: 7},
	)
	service := handover.NewService(store, certificationChecker{rejectedID: "module-b"}, func() time.Time { return fixed })

	err := service.Complete(context.Background(), "tenant-site-a", "batch-42")
	if !errors.Is(err, handover.ErrCertificationRejected) {
		t.Fatalf("Complete() error = %v, want certification rejection", err)
	}
	batch, modules, audits, ok := store.Snapshot("batch-42")
	if !ok {
		t.Fatal("batch disappeared after rejected handover")
	}
	if batch.Status != handover.BatchPending || !batch.CompletedAt.IsZero() {
		t.Fatalf("rejected batch = %#v, want pending with no completion time", batch)
	}
	wantRevisions := map[string]int{"module-a": 4, "module-b": 7}
	for _, module := range modules {
		if module.Status != handover.ModuleStaged || module.Revision != wantRevisions[module.ID] {
			t.Errorf("module %s after rejection = %#v, want staged at revision %d", module.ID, module, wantRevisions[module.ID])
		}
	}
	if len(audits) != 0 {
		t.Errorf("rejected handover wrote audits: %#v", audits)
	}

	validStore := handover.NewMemoryStore()
	validStore.Seed(
		handover.Batch{ID: "batch-43", TenantID: "tenant-site-a", ModuleIDs: []string{"module-c", "module-d"}, Status: handover.BatchPending},
		handover.Module{ID: "module-c", BatchID: "batch-43", Status: handover.ModuleStaged, Revision: 1},
		handover.Module{ID: "module-d", BatchID: "batch-43", Status: handover.ModuleStaged, Revision: 2},
	)
	validService := handover.NewService(validStore, certificationChecker{}, func() time.Time { return fixed })
	if err := validService.Complete(context.Background(), "tenant-site-a", "batch-43"); err != nil {
		t.Fatalf("valid Complete() error = %v", err)
	}
	validBatch, validModules, validAudits, ok := validStore.Snapshot("batch-43")
	if !ok || validBatch.Status != handover.BatchCompleted || !validBatch.CompletedAt.Equal(fixed) {
		t.Fatalf("valid batch = %#v, ok=%v", validBatch, ok)
	}
	for _, module := range validModules {
		if module.Status != handover.ModuleHandedOver {
			t.Errorf("valid module %s status = %s", module.ID, module.Status)
		}
	}
	if len(validAudits) != 2 || validAudits[0].ModuleID != "module-c" || validAudits[1].ModuleID != "module-d" {
		t.Fatalf("valid audits = %#v, want one ordered entry per module", validAudits)
	}
}
