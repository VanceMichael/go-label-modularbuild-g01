package settlement

import (
	"context"
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

	var receipt Receipt
	err := retry.Do(ctx, p.policy, func(attemptCtx context.Context, _ int) error {
		captured, err := p.gateway.Capture(attemptCtx, request.ChargeID, request.Amount, request.Currency)
		if err != nil {
			return fmt.Errorf("capture settlement: %w", err)
		}
		if captured.ID == "" {
			return domain.ErrInvalid
		}
		receipt = captured
		entry := finance.Entry{
			ID:           "settlement-" + captured.ID,
			TenantID:     request.TenantID,
			ModuleMoveID: request.ModuleMoveID,
			Currency:     request.Currency,
			Debit:        request.Amount,
			Memo:         "module movement settlement",
			PostedAt:     request.PostedAt,
		}
		if err := p.ledger.Post(entry); err != nil {
			return fmt.Errorf("post settlement: %w", err)
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return Result{ReceiptID: receipt.ID}, nil
}
