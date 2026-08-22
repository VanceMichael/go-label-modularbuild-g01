package loadcalculation

import "context"

type Request struct {
	PlanID   string
	Revision int
}
type Result struct {
	PlanID string
	PeakKG int
}
type Handler interface {
	Calculate(context.Context, Request) (Result, error)
}
