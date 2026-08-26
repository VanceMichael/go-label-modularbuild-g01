package liftpackagesign_test

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/liftpackagesign"
)

type observedPool struct {
	buffers     chan *bytes.Buffer
	waiting     chan struct{}
	waitingOnce sync.Once
}

func newObservedPool() *observedPool {
	pool := &observedPool{
		buffers: make(chan *bytes.Buffer, 1),
		waiting: make(chan struct{}),
	}
	pool.buffers <- &bytes.Buffer{}
	return pool
}

func (p *observedPool) Acquire(ctx context.Context) (*bytes.Buffer, error) {
	select {
	case buffer := <-p.buffers:
		return buffer, nil
	default:
		p.waitingOnce.Do(func() { close(p.waiting) })
	}
	select {
	case buffer := <-p.buffers:
		return buffer, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *observedPool) Release(buffer *bytes.Buffer) {
	buffer.Reset()
	p.buffers <- buffer
}

func (p *observedPool) Available() int {
	return len(p.buffers)
}

type delayedSigner struct {
	started map[string]chan struct{}
	allow   map[string]<-chan struct{}
}

func (s *delayedSigner) Start(_ context.Context, planID string, payload []byte) (liftpackagesign.Pending, error) {
	if started := s.started[planID]; started != nil {
		close(started)
	}
	return &pendingSignature{
		planID:  planID,
		payload: payload,
		allow:   s.allow[planID],
	}, nil
}

type pendingSignature struct {
	planID  string
	payload []byte
	allow   <-chan struct{}
}

func (p *pendingSignature) Wait(ctx context.Context) (liftpackagesign.SignedArtifact, error) {
	select {
	case <-p.allow:
		return liftpackagesign.SignedArtifact{
			PlanID:   p.planID,
			Payload:  append([]byte(nil), p.payload...),
			SignerID: "site-signer",
		}, nil
	case <-ctx.Done():
		return liftpackagesign.SignedArtifact{}, ctx.Err()
	}
}

type signResult struct {
	artifact liftpackagesign.SignedArtifact
	err      error
}

func TestConcurrentSigningKeepsPackageOwnershipUntilConsumed(t *testing.T) {
	pool := newObservedPool()
	startA := make(chan struct{})
	startB := make(chan struct{})
	allowA := make(chan struct{})
	immediate := make(chan struct{})
	close(immediate)
	signer := &delayedSigner{
		started: map[string]chan struct{}{
			"lift-plan-a": startA,
			"lift-plan-b": startB,
		},
		allow: map[string]<-chan struct{}{
			"lift-plan-a": allowA,
			"lift-plan-b": immediate,
			"lift-plan-c": immediate,
		},
	}
	service := liftpackagesign.NewService(pool, signer)
	issued := time.Date(2026, time.August, 22, 10, 0, 0, 0, time.UTC)

	firstDone := make(chan signResult, 1)
	go func() {
		artifact, err := service.RenderAndSign(context.Background(), liftpackagesign.Package{
			TenantID: "tenant-a",
			PlanID:   "lift-plan-a",
			Modules:  []string{"module-a1", "module-a2"},
			IssuedAt: issued,
		})
		firstDone <- signResult{artifact: artifact, err: err}
	}()
	<-startA

	secondDone := make(chan signResult, 1)
	go func() {
		artifact, err := service.RenderAndSign(context.Background(), liftpackagesign.Package{
			TenantID: "tenant-a",
			PlanID:   "lift-plan-b",
			Modules:  []string{"module-b1", "module-b2"},
			IssuedAt: issued.Add(time.Minute),
		})
		secondDone <- signResult{artifact: artifact, err: err}
	}()

	select {
	case <-startB:
		second := <-secondDone
		if second.err != nil {
			t.Fatalf("second signing error=%v", second.err)
		}
		secondDone <- second
	case <-pool.waiting:
	}
	close(allowA)

	first := <-firstDone
	second := <-secondDone
	if first.err != nil || second.err != nil {
		t.Fatalf("signing errors: first=%v second=%v", first.err, second.err)
	}
	if !bytes.Contains(first.artifact.Payload, []byte(`"plan_id":"lift-plan-a"`)) ||
		bytes.Contains(first.artifact.Payload, []byte(`"plan_id":"lift-plan-b"`)) {
		t.Errorf("first signed payload=%q", first.artifact.Payload)
	}
	if !bytes.Contains(second.artifact.Payload, []byte(`"plan_id":"lift-plan-b"`)) {
		t.Errorf("second signed payload=%q", second.artifact.Payload)
	}
	if pool.Available() != 1 {
		t.Errorf("available signing buffers=%d", pool.Available())
	}

	third, err := service.RenderAndSign(context.Background(), liftpackagesign.Package{
		TenantID: "tenant-a",
		PlanID:   "lift-plan-c",
		Modules:  []string{"module-c1"},
		IssuedAt: issued.Add(2 * time.Minute),
	})
	if err != nil || !bytes.Contains(third.Payload, []byte(`"plan_id":"lift-plan-c"`)) {
		t.Fatalf("third signing artifact=%#v error=%v", third, err)
	}
}
