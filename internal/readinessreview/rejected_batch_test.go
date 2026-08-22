package readinessreview_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/readinessreview"
)

type inspectionPlan struct {
	started chan struct{}
	release <-chan struct{}
	err     error
}

type controlledInspector struct {
	plans map[string]inspectionPlan
}

func (i *controlledInspector) Inspect(_ context.Context, moduleID string) error {
	plan := i.plans[moduleID]
	if plan.started != nil {
		close(plan.started)
	}
	if plan.release != nil {
		<-plan.release
	}
	return plan.err
}

type inspectorFunc func(context.Context, string) error

func (f inspectorFunc) Inspect(ctx context.Context, moduleID string) error {
	return f(ctx, moduleID)
}

type notifyingRecorder struct {
	store    *readinessreview.Store
	recorded chan readinessreview.Result
}

func (r *notifyingRecorder) Record(ctx context.Context, result readinessreview.Result) error {
	if err := r.store.Record(ctx, result); err != nil {
		return err
	}
	r.recorded <- result
	return nil
}

func TestRejectedBatchDoesNotPublishLateReadiness(t *testing.T) {
	moduleAStarted := make(chan struct{})
	moduleBStarted := make(chan struct{})
	releaseModuleA := make(chan struct{})
	store := readinessreview.NewStore()
	recorder := &notifyingRecorder{
		store:    store,
		recorded: make(chan readinessreview.Result, 1),
	}
	inspector := &controlledInspector{plans: map[string]inspectionPlan{
		"module-a": {started: moduleAStarted, release: releaseModuleA},
		"module-b": {started: moduleBStarted, err: readinessreview.ErrInspectionRejected},
	}}
	service := readinessreview.NewService(inspector, recorder, func() time.Time {
		return time.Date(2026, time.August, 22, 9, 0, 0, 0, time.UTC)
	})

	returned := make(chan error, 1)
	go func() {
		returned <- service.ReviewBatch(
			context.Background(),
			"tenant-a",
			"readiness-42",
			[]string{"module-a", "module-b"},
		)
	}()
	<-moduleAStarted
	<-moduleBStarted
	if err := <-returned; !errors.Is(err, readinessreview.ErrInspectionRejected) {
		t.Fatalf("ReviewBatch error=%v", err)
	}
	if got := store.Results("tenant-a", "readiness-42"); len(got) != 0 {
		t.Fatalf("rejected batch had results before release: %#v", got)
	}

	close(releaseModuleA)
	select {
	case result := <-recorder.recorded:
		if result.ModuleID != "module-a" || result.Status != "ready" {
			t.Errorf("late result=%#v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("unfinished inspection did not reach recorder")
	}
	if got := store.Results("tenant-a", "readiness-42"); len(got) != 0 {
		t.Errorf("rejected batch published late results: %#v", got)
	}

	validStore := readinessreview.NewStore()
	validService := readinessreview.NewService(
		inspectorFunc(func(context.Context, string) error { return nil }),
		validStore,
		nil,
	)
	if err := validService.ReviewBatch(
		context.Background(),
		"tenant-a",
		"readiness-43",
		[]string{"module-c", "module-d"},
	); err != nil {
		t.Fatalf("valid ReviewBatch error=%v", err)
	}
	valid := validStore.Results("tenant-a", "readiness-43")
	if len(valid) != 2 || valid[0].ModuleID != "module-c" || valid[1].ModuleID != "module-d" {
		t.Fatalf("valid results=%#v", valid)
	}
}
