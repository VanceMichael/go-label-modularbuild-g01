package cranevalidation_test

import (
	"context"
	"errors"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/cranevalidation"
	"testing"
)

type validator struct {
	started chan struct{}
	block   bool
}

func (v *validator) Validate(ctx context.Context, r cranevalidation.Request) (cranevalidation.Result, error) {
	if v.block {
		v.block = false
		close(v.started)
		<-ctx.Done()
		return cranevalidation.Result{}, ctx.Err()
	}
	return cranevalidation.Result{CraneID: r.CraneID, Valid: true}, nil
}
func TestCanceledValidationReleasesCraneLease(t *testing.T) {
	leasing := cranevalidation.NewLeases()
	v := &validator{started: make(chan struct{}), block: true}
	service := cranevalidation.NewService(leasing, v)
	request := cranevalidation.Request{TenantID: "tenant-a", CraneID: "crane-12", ConfigVersion: "v7"}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := service.Run(ctx, request); done <- err }()
	<-v.started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("canceled Run error=%v", err)
	}
	if leasing.Held("tenant-a/crane-12") {
		t.Errorf("canceled validation retained lease")
	}
	result, err := service.Run(context.Background(), request)
	if err != nil || !result.Valid || result.CraneID != "crane-12" {
		t.Errorf("retry result=%#v error=%v", result, err)
	}
	if leasing.Held("tenant-a/crane-12") {
		t.Errorf("retry retained lease")
	}
	cleanLeases := cranevalidation.NewLeases()
	clean := cranevalidation.NewService(cleanLeases, &validator{started: make(chan struct{})})
	result, err = clean.Run(context.Background(), cranevalidation.Request{TenantID: "tenant-b", CraneID: "crane-13", ConfigVersion: "v1"})
	if err != nil || !result.Valid || cleanLeases.Held("tenant-b/crane-13") {
		t.Fatalf("clean result=%#v error=%v held=%v", result, err, cleanLeases.Held("tenant-b/crane-13"))
	}
}
