package settlement

import (
	"context"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/finance"
)

type Request struct {
	ChargeID     string
	TenantID     string
	ModuleMoveID string
	Amount       int64
	Currency     string
	PostedAt     time.Time
}

type Receipt struct {
	ID string
}

type Result struct {
	ReceiptID string
}

type Gateway interface {
	Capture(context.Context, string, int64, string) (Receipt, error)
}

type Ledger interface {
	Post(finance.Entry) error
}
