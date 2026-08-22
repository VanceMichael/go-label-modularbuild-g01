package eta

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"time"
)

type Stop struct {
	Code   string
	Arrive time.Time
	Depart time.Time
}
type Estimate struct {
	ModuleMoveID string
	Arrival      time.Time
	Confidence   string
	DelayMinutes int
}

func Calculate(module_move string, stops []Stop, now time.Time) (Estimate, error) {
	if module_move == "" || len(stops) == 0 {
		return Estimate{}, domain.ErrInvalid
	}
	last := now
	for _, s := range stops {
		if s.Code == "" || !s.Depart.After(s.Arrive) || s.Arrive.Before(last) {
			return Estimate{}, domain.ErrInvalid
		}
		last = s.Depart
	}
	delay := 0
	if last.Before(now) {
		delay = int(now.Sub(last).Minutes())
	}
	confidence := "high"
	if len(stops) > 3 {
		confidence = "medium"
	}
	return Estimate{ModuleMoveID: module_move, Arrival: last, Confidence: confidence, DelayMinutes: delay}, nil
}
func AddDelay(e Estimate, minutes int) (Estimate, error) {
	if minutes < 0 {
		return Estimate{}, domain.ErrInvalid
	}
	e.Arrival = e.Arrival.Add(time.Duration(minutes) * time.Minute)
	e.DelayMinutes += minutes
	if e.DelayMinutes > 120 {
		e.Confidence = "low"
	}
	return e, nil
}
