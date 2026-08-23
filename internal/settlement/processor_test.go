package settlement

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/finance"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/retry"
)

// The live document under repair carries the business label:
// charge-14, tenant-1, module-14, CNY, receipt-charge-14-1.
// The fakes below reproduce its three processing paths.
type fakeGateway struct {
	mu       sync.Mutex
	captures int
	receipt  Receipt
	err      error
}

func (g *fakeGateway) Capture(_ context.Context, _ string, _ int64, _ string) (Receipt, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.captures++
	if g.err != nil {
		return Receipt{}, g.err
	}
	return g.receipt, nil
}

func newRequest(now time.Time) Request {
	return Request{
		ChargeID:     "charge-14",
		TenantID:     "tenant-1",
		ModuleMoveID: "module-14",
		Amount:       4800,
		Currency:     "CNY",
		PostedAt:     now,
	}
}

func quickPolicy() retry.Policy {
	return retry.Policy{MaxAttempts: 5, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond}
}

// TestSettleNoFault covers the normal path: capture once, post once.
func TestSettleNoFault(t *testing.T) {
	now := time.Now().UTC()
	gw := &fakeGateway{receipt: Receipt{ID: "receipt-charge-14-1"}}
	led := finance.New()
	p := NewProcessor(gw, led, quickPolicy())

	res, err := p.Settle(context.Background(), newRequest(now))
	if err != nil {
		t.Fatal(err)
	}
	if res.ReceiptID != "receipt-charge-14-1" {
		t.Fatalf("receipt id = %q", res.ReceiptID)
	}
	if gw.captures != 1 {
		t.Fatalf("capture count = %d, want 1", gw.captures)
	}
	entries := led.Entries("tenant-1")
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	e := entries[0]
	if e.ID != "settlement-receipt-charge-14-1" || e.TenantID != "tenant-1" || e.ModuleMoveID != "module-14" ||
		e.Currency != "CNY" || e.Debit != 4800 {
		t.Fatalf("entry = %#v", e)
	}
	if err := led.Reconcile("tenant-1"); err != nil {
		t.Fatal(err)
	}
}

// flakyLedger returns a transient error on the first Post and commits on the
// second. This models the field incident: the first accounting Post returned a
// temporary error, and recovery on the second attempt committed the record.
type flakyLedger struct {
	inner   *finance.Ledger
	failure error
	posts   int
}

func (f *flakyLedger) Post(e finance.Entry) error {
	f.posts++
	if f.posts == 1 {
		return f.failure
	}
	return f.inner.Post(e)
}

// TestSettleTransientPostRecovers covers the recovery path: first Post fails
// transiently, second attempt commits. Capture runs exactly once and the
// result reuses the original capture receipt.
func TestSettleTransientPostRecovers(t *testing.T) {
	now := time.Now().UTC()
	gw := &fakeGateway{receipt: Receipt{ID: "receipt-charge-14-1"}}
	led := finance.New()
	flaky := &flakyLedger{inner: led, failure: errors.New("temporary accounting failure")}
	p := NewProcessor(gw, flaky, quickPolicy())

	res, err := p.Settle(context.Background(), newRequest(now))
	if err != nil {
		t.Fatal(err)
	}
	if res.ReceiptID != "receipt-charge-14-1" {
		t.Fatalf("receipt id = %q, want original capture receipt", res.ReceiptID)
	}
	if gw.captures != 1 {
		t.Fatalf("capture count = %d, want 1 (capture must not be retried)", gw.captures)
	}
	if flaky.posts != 2 {
		t.Fatalf("post count = %d, want 2", flaky.posts)
	}
	entries := led.Entries("tenant-1")
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want exactly one debit", len(entries))
	}
	e := entries[0]
	if e.ID != "settlement-receipt-charge-14-1" || e.Debit != 4800 || e.Currency != "CNY" {
		t.Fatalf("entry = %#v", e)
	}
	if err := led.Reconcile("tenant-1"); err != nil {
		t.Fatal(err)
	}
}

// reconciledLedger commits on the first Post but reports a conflict on replay.
// This models the failed wind-down path: the record was already persisted by a
// prior attempt, so the retry must reconcile to success rather than double-post.
type reconciledLedger struct{ inner *finance.Ledger }

func (r *reconciledLedger) Post(e finance.Entry) error {
	if err := r.inner.Post(e); err != nil {
		return err
	}
	return r.inner.Post(e) // already present: returns domain.ErrConflict
}

// TestSettleAlreadyPostedReconciles covers the failed wind-down path: the
// ledger already holds the debit, the replay conflicts, and Settle treats the
// conflict as idempotent success while reusing the original receipt.
func TestSettleAlreadyPostedReconciles(t *testing.T) {
	now := time.Now().UTC()
	gw := &fakeGateway{receipt: Receipt{ID: "receipt-charge-14-1"}}
	led := finance.New()
	rec := &reconciledLedger{inner: led}
	p := NewProcessor(gw, rec, quickPolicy())

	res, err := p.Settle(context.Background(), newRequest(now))
	if err != nil {
		t.Fatal(err)
	}
	if res.ReceiptID != "receipt-charge-14-1" {
		t.Fatalf("receipt id = %q, want original capture receipt", res.ReceiptID)
	}
	if gw.captures != 1 {
		t.Fatalf("capture count = %d, want 1", gw.captures)
	}
	entries := led.Entries("tenant-1")
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want exactly one debit", len(entries))
	}
	if entries[0].Debit != 4800 || entries[0].Currency != "CNY" {
		t.Fatalf("entry = %#v", entries[0])
	}
	if err := led.Reconcile("tenant-1"); err != nil {
		t.Fatal(err)
	}
}

// TestSettlePermanentCaptureFailure ensures a capture failure is surfaced
// without touching the ledger.
func TestSettlePermanentCaptureFailure(t *testing.T) {
	now := time.Time{}.Add(time.Second)
	gw := &fakeGateway{err: errors.New("gateway unavailable")}
	led := finance.New()
	p := NewProcessor(gw, led, quickPolicy())

	if _, err := p.Settle(context.Background(), newRequest(now)); err == nil {
		t.Fatal("expected capture error")
	}
	if gw.captures != 1 {
		t.Fatalf("capture count = %d, want 1", gw.captures)
	}
	if len(led.Entries("tenant-1")) != 0 {
		t.Fatal("ledger must remain untouched on capture failure")
	}
}

// TestSettleInvalidRequest guards the validation gate.
func TestSettleInvalidRequest(t *testing.T) {
	gw := &fakeGateway{receipt: Receipt{ID: "receipt-charge-14-1"}}
	led := finance.New()
	p := NewProcessor(gw, led, quickPolicy())
	now := time.Now().UTC()
	cases := []Request{
		{}, // all blank
		{ChargeID: "charge-14", TenantID: "tenant-1", ModuleMoveID: "module-14", Amount: 0, Currency: "CNY", PostedAt: now},
		{ChargeID: "charge-14", TenantID: "tenant-1", ModuleMoveID: "module-14", Amount: 4800, Currency: "", PostedAt: now},
		{ChargeID: "charge-14", TenantID: "tenant-1", ModuleMoveID: "module-14", Amount: 4800, Currency: "CNY"},
	}
	for i, c := range cases {
		if _, err := p.Settle(context.Background(), c); err != domain.ErrInvalid {
			t.Fatalf("case %d: err = %v, want ErrInvalid", i, err)
		}
	}
	if gw.captures != 0 {
		t.Fatalf("capture count = %d, want 0 on invalid request", gw.captures)
	}
}
