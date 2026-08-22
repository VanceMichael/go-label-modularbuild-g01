package liftmethod

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

func TestConcurrentRevisionsRejectStaleVersion(t *testing.T) {
	ctx := context.Background()
	store := NewMemory()
	if err := store.Add(Statement{ID: "method-25", TenantID: "tenant-1", Title: "Initial lift", Revision: "R1"}); err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	review := ReviewFunc(func(context.Context, Statement) error {
		ready <- struct{}{}
		<-release
		return nil
	})
	service := NewService(store, review).WithClock(func() time.Time {
		return time.Date(2026, 8, 22, 20, 0, 0, 0, time.UTC)
	})
	results := make(chan error, 2)
	go func() {
		results <- service.Revise(ctx, "tenant-1", "method-25", "North lift", "R2-N", "supervisor-n", 1)
	}()
	go func() {
		results <- service.Revise(ctx, "tenant-1", "method-25", "South lift", "R2-S", "supervisor-s", 1)
	}()
	<-ready
	<-ready
	close(release)

	successes := 0
	conflicts := 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrConflict):
			conflicts++
		default:
			t.Errorf("concurrent revision error = %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Errorf("concurrent revision results: successes=%d conflicts=%d", successes, conflicts)
	}
	statement, err := store.Get(ctx, "tenant-1", "method-25")
	if err != nil || statement.Version != 2 || (statement.Revision != "R2-N" && statement.Revision != "R2-S") {
		t.Errorf("statement after concurrent revisions = %+v, err=%v", statement, err)
	}
	if count := store.AuditCount(); count != 1 {
		t.Errorf("audit count after concurrent revisions = %d, want 1", count)
	}

	sequential := NewService(store, ReviewFunc(func(context.Context, Statement) error { return nil })).
		WithClock(func() time.Time { return time.Date(2026, 8, 22, 20, 5, 0, 0, time.UTC) })
	if err := sequential.Revise(ctx, "tenant-1", "method-25", "Approved lift", "R3", "chief-engineer", 2); err != nil {
		t.Fatalf("sequential revision from version 2: %v", err)
	}
	statement, err = store.Get(ctx, "tenant-1", "method-25")
	if err != nil || statement.Version != 3 || statement.Revision != "R3" || statement.UpdatedBy != "chief-engineer" {
		t.Errorf("statement after sequential revision = %+v, err=%v", statement, err)
	}
	if count := store.AuditCount(); count != 2 {
		t.Errorf("audit count after sequential revision = %d, want 2", count)
	}
}
