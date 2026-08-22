package manifestimport

import (
	"context"
	"errors"
	"testing"
)

func TestReadFailureReleasesManifestForImmediateRetry(t *testing.T) {
	ctx := context.Background()
	source := NewMemorySource()
	rows := []Row{
		{ID: "row-1", TenantID: "tenant-1", ModuleID: "module-21-a", WeightKg: 4200},
		{ID: "row-2", TenantID: "tenant-1", ModuleID: "module-21-b", WeightKg: 3800},
	}
	if err := source.AddFile("tenant-1", "manifest-21", rows, 1); err != nil {
		t.Fatal(err)
	}
	store := NewMemoryStore()
	importer := NewImporter(source, store)

	err := importer.Import(ctx, "tenant-1", "manifest-21")
	if !errors.Is(err, ErrReadFault) {
		t.Fatalf("first import error = %v, want read fault", err)
	}
	persisted, err := store.Rows(ctx, "tenant-1")
	if err != nil || len(persisted) != 0 {
		t.Errorf("rows after failed import = %+v, err=%v", persisted, err)
	}
	if count := source.OpenCount(); count != 0 {
		t.Errorf("open manifest sources after failed import = %d, want 0", count)
	}

	if err := importer.Import(ctx, "tenant-1", "manifest-21"); err != nil {
		t.Fatalf("immediate retry: %v", err)
	}
	persisted, err = store.Rows(ctx, "tenant-1")
	if err != nil || len(persisted) != 2 || persisted[0] != rows[0] || persisted[1] != rows[1] {
		t.Errorf("rows after successful retry = %+v, err=%v", persisted, err)
	}
	if count := source.OpenCount(); count != 0 {
		t.Errorf("open manifest sources after successful retry = %d, want 0", count)
	}
}
