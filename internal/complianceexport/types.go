package complianceexport

import "context"

type Module struct {
	ID       string
	TenantID string
	Serial   string
	SiteCode string
}

type Signer interface {
	Sign(context.Context, Module) (string, error)
}
