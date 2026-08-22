package liftanalysis

import "context"

type Request struct {
	TenantID string
	PlanID   string
	Revision int
}

type Result struct {
	PlanID string
	Safe   bool
}

type Engine interface {
	Analyze(context.Context, Request) (Result, error)
}
