package settlement

import (
	"context"
	"errors"
	"fmt"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/finance"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/retry"
)

type Processor struct {
	gateway Gateway
	ledger  Ledger
	policy  retry.Policy
}

func NewProcessor(gateway Gateway, ledger Ledger, policy retry.Policy) *Processor {
	return &Processor{gateway: gateway, ledger: ledger, policy: policy}
}

func (p *Processor) Settle(ctx context.Context, request Request) (Result, error) {
	if p.gateway == nil || p.ledger == nil || request.ChargeID == "" || request.TenantID == "" ||
		request.ModuleMoveID == "" || request.Amount <= 0 || request.Currency == "" || request.PostedAt.IsZero() {
		return Result{}, domain.ErrInvalid
	}

	// Capture exactly once, outside the retry loop. A transient Ledger.Post
	// failure must never trigger a second capture; the original capture receipt
	// is reused for every subsequent ledger attempt and as the final result.
	captured, err := p.gateway.Capture(ctx, request.ChargeID, request.Amount, request.Currency)
	if err != nil {
		return Result{}, fmt.Errorf("capture settlement: %w", err)
	}
	if captured.ID == "" {
		return Result{}, domain.ErrInvalid
	}

	entry := finance.Entry{
		ID:           "settlement-" + captured.ID,
		TenantID:     request.TenantID,
		ModuleMoveID: request.ModuleMoveID,
		Currency:     request.Currency,
		Debit:        request.Amount,
		Memo:         "module movement settlement",
		PostedAt:     request.PostedAt,
	}

	// Retry only the ledger post. The entry ID is derived from the single
	// capture receipt, so a replay that finds an already-persisted entry is a
	// conflict — which means the prior attempt committed the record and is
	// treated as idempotent success rather than a failure.
	err = retry.Do(ctx, p.policy, func(attemptCtx context.Context, _ int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if postErr := p.ledger.Post(entry); postErr != nil {
			if errors.Is(postErr, domain.ErrConflict) {
				return nil
			}
			return fmt.Errorf("post settlement: %w", postErr)
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return Result{ReceiptID: captured.ID}, nil
}
