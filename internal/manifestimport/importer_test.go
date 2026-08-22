package manifestimport

import (
	"context"
	"errors"
	"testing"
)

// Reproduces the field scenario for tenant-1 / manifest-21:
//   row-1 -> module-21-a (valid)
//   row-2 -> module-21-b (valid payload, but the source fails on the second read)
//
// On the first import, when ErrReadFault is hit at the second row, the import
// must keep zero rows persisted and restore the source open count to 0.
func TestImportReadFaultOnSecondRowKeepsZeroRowsAndClosedSource(t *testing.T) {
	tenant := "tenant-1"
	fileID := "manifest-21"
	rows := []Row{
		{ID: "row-1", TenantID: tenant, ModuleID: "module-21-a", WeightKg: 1200},
		{ID: "row-2", TenantID: tenant, ModuleID: "module-21-b", WeightKg: 980},
	}

	source := NewMemorySource()
	if err := source.AddFile(tenant, fileID, rows, 1); err != nil { // failOnceAt == 1
		t.Fatalf("AddFile: %v", err)
	}
	store := NewMemoryStore()
	importer := NewImporter(source, store)

	err := importer.Import(context.Background(), tenant, fileID)
	if !errors.Is(err, ErrReadFault) {
		t.Fatalf("expected ErrReadFault, got %v", err)
	}

	if got := source.OpenCount(); got != 0 {
		t.Fatalf("source open count = %d, want 0 (source must be closed after failure)", got)
	}
	persisted, err := store.Rows(context.Background(), tenant)
	if err != nil {
		t.Fatalf("Rows: %v", err)
	}
	if len(persisted) != 0 {
		t.Fatalf("persisted rows = %d, want 0 (transaction must roll back row-1)", len(persisted))
	}
}
