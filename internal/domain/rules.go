package domain

import "time"

func (s ModuleMoveStatus) CanTransition(next ModuleMoveStatus) bool {
	allowed := map[ModuleMoveStatus]map[ModuleMoveStatus]bool{
		ModuleMoveDraft:     {ModuleMoveBooked: true, ModuleMoveCancelled: true},
		ModuleMoveBooked:    {ModuleMoveScreening: true, ModuleMoveCancelled: true},
		ModuleMoveScreening: {ModuleMoveCleared: true, ModuleMoveCancelled: true},
		ModuleMoveCleared:   {ModuleMoveLoaded: true, ModuleMoveCancelled: true},
		ModuleMoveLoaded:    {ModuleMoveDeparted: true}, ModuleMoveDeparted: {}, ModuleMoveCancelled: {},
	}
	return allowed[s][next]
}

func (s ModuleMove) CanTransition(next ModuleMoveStatus) bool { return s.Status.CanTransition(next) }

func (s WindowStatus) CanTransition(next WindowStatus) bool {
	return map[WindowStatus]map[WindowStatus]bool{WindowPlanned: {WindowOpen: true}, WindowOpen: {WindowBoarding: true, WindowClosed: true}, WindowBoarding: {WindowDeparted: true, WindowClosed: true}, WindowDeparted: {WindowClosed: true}, WindowClosed: {}}[s][next]
}
func (s QualityStatus) CanTransition(next QualityStatus) bool {
	return map[QualityStatus]map[QualityStatus]bool{QualityPending: {QualityReview: true, QualityHeld: true}, QualityReview: {QualityReleased: true, QualityHeld: true}, QualityHeld: {QualityReview: true}, QualityReleased: {}}[s][next]
}
func (s SiteSafetyStatus) CanTransition(next SiteSafetyStatus) bool {
	return map[SiteSafetyStatus]map[SiteSafetyStatus]bool{SiteSafetyPending: {SiteSafetyPassed: true, SiteSafetyFailed: true}, SiteSafetyFailed: {SiteSafetyPending: true}, SiteSafetyPassed: {}}[s][next]
}

func ValidateModuleMove(s ModuleMove) error {
	if s.TenantID == "" || s.Reference == "" || s.Origin == "" || s.Destination == "" || s.WeightKg <= 0 || s.Pieces <= 0 {
		return ErrInvalid
	}
	if s.Origin == s.Destination {
		return ErrInvalid
	}
	return nil
}
func ValidateLeg(l LiftWindow, now time.Time) error {
	if l.TenantID == "" || l.LiftNumber == "" || l.Origin == "" || l.Destination == "" || l.CapacityKg <= 0 || !l.DepartureAt.After(now) || !l.ArrivalAt.After(l.DepartureAt) {
		return ErrInvalid
	}
	return nil
}
func ValidateCapacity(l LiftWindow, weight int64) error {
	if weight <= 0 || l.ReservedKg < 0 || l.ReservedKg+weight > l.CapacityKg {
		return ErrCapacity
	}
	return nil
}
func QualityAllowsLoading(c QualityCase) bool        { return c.Status == QualityReleased }
func SiteSafetyAllowsLoading(s SiteSafetyCheck) bool { return s.Status == SiteSafetyPassed }
func IsSessionActive(s Session, now time.Time) error {
	if s.RevokedAt != nil {
		return ErrRevoked
	}
	if !now.Before(s.ExpiresAt) {
		return ErrExpired
	}
	return nil
}
