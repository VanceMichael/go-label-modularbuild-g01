package settlement

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/finance"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/retry"
)

type recordingGateway struct {
	captures int
}

func (g *recordingGateway) Capture(_ context.Context, chargeID string, _ int64, _ string) (Receipt, error) {
	g.captures++
	return Receipt{ID: fmt.Sprintf("receipt-%s-%d", chargeID, g.captures)}, nil
}

type transientLedger struct {
	failures int
	attempts int
	entries  []finance.Entry
}

func (l *transientLedger) Post(entry finance.Entry) error {
	l.attempts++
	if l.attempts <= l.failures {
		return errors.New("ledger temporarily unavailable")
	}
	l.entries = append(l.entries, entry)
	return nil
}

func TestSettlementLedgerRetryDoesNotRepeatCapture(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	request := Request{
		ChargeID:     "charge-14",
		TenantID:     "tenant-1",
		ModuleMoveID: "module-14",
		Amount:       4800,
		Currency:     "CNY",
		PostedAt:     now,
	}
	policy := retry.Policy{MaxAttempts: 2, InitialDelay: time.Nanosecond, MaxDelay: time.Nanosecond}

	t.Run("temporary ledger failure reuses the completed capture", func(t *testing.T) {
		gateway := &recordingGateway{}
		ledger := &transientLedger{failures: 1}
		result, err := NewProcessor(gateway, ledger, policy).Settle(context.Background(), request)
		if err != nil {
			t.Fatalf("settlement after ledger recovery: %v", err)
		}
		if gateway.captures != 1 {
			t.Errorf("gateway captures = %d, want 1", gateway.captures)
		}
		if ledger.attempts != 2 || len(ledger.entries) != 1 {
			t.Errorf("ledger activity: attempts=%d entries=%d, want 2 attempts and 1 entry", ledger.attempts, len(ledger.entries))
		}
		if result.ReceiptID != "receipt-charge-14-1" {
			t.Errorf("settlement receipt = %q, want first capture receipt", result.ReceiptID)
		}
		if len(ledger.entries) == 1 {
			entry := ledger.entries[0]
			if entry.TenantID != request.TenantID || entry.ModuleMoveID != request.ModuleMoveID || entry.Debit != request.Amount || entry.Currency != request.Currency {
				t.Errorf("settlement ledger entry = %+v", entry)
			}
		}
	})

	t.Run("ordinary settlement captures and posts once", func(t *testing.T) {
		gateway := &recordingGateway{}
		ledger := &transientLedger{}
		result, err := NewProcessor(gateway, ledger, policy).Settle(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if gateway.captures != 1 || ledger.attempts != 1 || len(ledger.entries) != 1 {
			t.Fatalf("ordinary settlement activity: captures=%d attempts=%d entries=%d", gateway.captures, ledger.attempts, len(ledger.entries))
		}
		if result.ReceiptID != "receipt-charge-14-1" {
			t.Fatalf("ordinary settlement receipt = %q", result.ReceiptID)
		}
	})
}
