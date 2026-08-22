package domain

import (
	"testing"
	"time"
)

func TestModuleMoveTransitions(t *testing.T) {
	cases := []struct {
		from ModuleMoveStatus
		to   ModuleMoveStatus
		ok   bool
	}{{ModuleMoveDraft, ModuleMoveBooked, true}, {ModuleMoveBooked, ModuleMoveScreening, true}, {ModuleMoveScreening, ModuleMoveCleared, true}, {ModuleMoveCleared, ModuleMoveLoaded, true}, {ModuleMoveLoaded, ModuleMoveDeparted, true}, {ModuleMoveDraft, ModuleMoveDeparted, false}, {ModuleMoveDeparted, ModuleMoveDraft, false}}
	for _, tc := range cases {
		if got := tc.from.CanTransition(tc.to); got != tc.ok {
			t.Fatalf("%s to %s got %v", tc.from, tc.to, got)
		}
	}
}
func TestLegTransitions(t *testing.T) {
	if !WindowPlanned.CanTransition(WindowOpen) {
		t.Fatal("planned should open")
	}
	if WindowOpen.CanTransition(WindowDeparted) {
		t.Fatal("open cannot depart")
	}
	if WindowClosed.CanTransition(WindowOpen) {
		t.Fatal("closed cannot reopen")
	}
}
func TestValidation(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := ValidateModuleMove(ModuleMove{TenantID: "t", Reference: "R", Origin: "PEK", Destination: "FRA", WeightKg: 10, Pieces: 1}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateModuleMove(ModuleMove{TenantID: "t", Reference: "R", Origin: "PEK", Destination: "PEK", WeightKg: 10, Pieces: 1}); err == nil {
		t.Fatal("same route should fail")
	}
	if err := ValidateLeg(LiftWindow{TenantID: "t", LiftNumber: "AB1", Origin: "PEK", Destination: "FRA", DepartureAt: now.Add(time.Hour), ArrivalAt: now.Add(2 * time.Hour), CapacityKg: 1}, now); err != nil {
		t.Fatal(err)
	}
}
func TestSessionBoundaries(t *testing.T) {
	now := time.Unix(100, 0)
	s := Session{ExpiresAt: now}
	if IsSessionActive(s, now) != ErrExpired {
		t.Fatal("expiry must be inclusive")
	}
	s.ExpiresAt = now.Add(time.Second)
	if IsSessionActive(s, now) != nil {
		t.Fatal("session should be active")
	}
}
