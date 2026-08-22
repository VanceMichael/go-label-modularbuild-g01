package transportquote

import "context"

type Request struct {
	TenantID, RouteID string
	WeightKG          int
}
type Quote struct {
	RouteID     string
	AmountCents int
}
type Provider interface {
	Quote(context.Context, Request) (Quote, error)
}
