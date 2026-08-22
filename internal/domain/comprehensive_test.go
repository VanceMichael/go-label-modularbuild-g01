package domain

import (
	"testing"
	"time"
)

func TestComprehensiveDomainCases(t *testing.T) {
	cases := []struct {
		name string
		from ModuleMoveStatus
		to   ModuleMoveStatus
		ok   bool
	}{
		{"draft book", ModuleMoveDraft, ModuleMoveBooked, true},
		{"draft cancel", ModuleMoveDraft, ModuleMoveCancelled, true},
		{"book screen", ModuleMoveBooked, ModuleMoveScreening, true},
		{"book cancel", ModuleMoveBooked, ModuleMoveCancelled, true},
		{"screen clear", ModuleMoveScreening, ModuleMoveCleared, true},
		{"screen hold", ModuleMoveScreening, ModuleMoveCancelled, true},
		{"clear load", ModuleMoveCleared, ModuleMoveLoaded, true},
		{"load depart", ModuleMoveLoaded, ModuleMoveDeparted, true},
		{"depart draft", ModuleMoveDeparted, ModuleMoveDraft, false},
		{"cancel book", ModuleMoveCancelled, ModuleMoveBooked, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.from.CanTransition(tc.to); got != tc.ok {
				t.Fatalf("got %v", got)
			}
		})
	}
}
func TestRulesRejectBadCapacity(t *testing.T) {
	window := LiftWindow{CapacityKg: 10, ReservedKg: 9}
	if err := ValidateCapacity(window, 1); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCapacity(window, 2); err != ErrCapacity {
		t.Fatal(err)
	}
	window.ReservedKg = -1
	if err := ValidateCapacity(window, 1); err != ErrCapacity {
		t.Fatal(err)
	}
}
func TestRulesStatusFamilies(t *testing.T) {
	if !QualityPending.CanTransition(QualityReview) {
		t.Fatal("quality pending")
	}
	if QualityReleased.CanTransition(QualityReview) {
		t.Fatal("quality released")
	}
	if !SiteSafetyPending.CanTransition(SiteSafetyPassed) {
		t.Fatal("site_safety pending")
	}
	if SiteSafetyPassed.CanTransition(SiteSafetyFailed) {
		t.Fatal("site_safety passed")
	}
}
func TestClockAndPagination(t *testing.T) {
	now := time.Now()
	clock := FixedClock{T: now}
	if !clock.Now().Equal(now) {
		t.Fatal("clock changed")
	}
	req := PageRequest{Limit: 1000}.Normalized()
	if req.Limit != 200 {
		t.Fatal(req.Limit)
	}
	req = PageRequest{Limit: 0}.Normalized()
	if req.Limit != 50 {
		t.Fatal(req.Limit)
	}
}
func TestValidationMessages(t *testing.T) {
	bad := []ModuleMove{{}, {TenantID: "t"}, {TenantID: "t", Reference: "r", Origin: "A", Destination: "B", WeightKg: -1, Pieces: 1}, {TenantID: "t", Reference: "r", Origin: "A", Destination: "B", WeightKg: 1, Pieces: 0}}
	for i, v := range bad {
		if err := ValidateModuleMove(v); err == nil {
			t.Fatalf("case %d accepted", i)
		}
	}
}
