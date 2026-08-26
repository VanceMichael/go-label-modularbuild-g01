package permit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

func TestFailedPermitApprovalDoesNotAuthorizeDispatch(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 22, 15, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	cache := NewMemoryCache()
	service := NewService(store, cache).WithClock(func() time.Time { return now })
	permit := Permit{ID: "permit-19", TenantID: "tenant-1", ModuleID: "module-19"}
	if err := store.Add(permit); err != nil {
		t.Fatal(err)
	}

	store.FailNextAudit()
	err := service.Approve(ctx, "tenant-1", permit.ID, "safety-manager")
	if !errors.Is(err, ErrAuditUnavailable) {
		t.Fatalf("failed approval error = %v, want audit unavailable", err)
	}
	persisted, err := store.GetPermit(ctx, "tenant-1", permit.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != StatusPending || persisted.Version != 1 || persisted.ApprovedBy != "" || persisted.ApprovedAt != nil {
		t.Errorf("permit after rolled back approval = %+v", persisted)
	}
	if cached, err := cache.Get(ctx, "tenant-1", permit.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("readiness cache after rolled back approval = %+v, err=%v", cached, err)
	}
	allowed, err := service.CanDispatch(ctx, "tenant-1", permit.ID)
	if err != nil || allowed {
		t.Errorf("dispatch after rolled back approval: allowed=%v err=%v", allowed, err)
	}
	if count := store.AuditCount(); count != 0 {
		t.Errorf("audit count after rolled back approval = %d, want 0", count)
	}

	if err := service.Approve(ctx, "tenant-1", permit.ID, "safety-manager"); err != nil {
		t.Fatalf("successful approval: %v", err)
	}
	persisted, err = store.GetPermit(ctx, "tenant-1", permit.ID)
	if err != nil || persisted.Status != StatusApproved || persisted.Version != 2 || persisted.ApprovedBy != "safety-manager" || persisted.ApprovedAt == nil || !persisted.ApprovedAt.Equal(now) {
		t.Errorf("permit after successful approval = %+v, err=%v", persisted, err)
	}
	cached, err := cache.Get(ctx, "tenant-1", permit.ID)
	if err != nil || cached.Status != StatusApproved || cached.Version != 2 {
		t.Errorf("readiness cache after successful approval = %+v, err=%v", cached, err)
	}
	allowed, err = service.CanDispatch(ctx, "tenant-1", permit.ID)
	if err != nil || !allowed {
		t.Errorf("dispatch after successful approval: allowed=%v err=%v", allowed, err)
	}
	if count := store.AuditCount(); count != 1 {
		t.Errorf("audit count after successful approval = %d, want 1", count)
	}
}
