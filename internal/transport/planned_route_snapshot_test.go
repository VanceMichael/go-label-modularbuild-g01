package transport

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/route"
)

func TestPlannedAssignmentOwnsItsRouteSnapshot(t *testing.T) {
	now := time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)
	planner := New()
	if err := planner.AddVehicle(Vehicle{
		ID:            "vehicle-1",
		Registration:  "SITE-LIFT-01",
		MaxKg:         20,
		AvailableFrom: now.Add(-time.Hour),
		AvailableTo:   now.Add(8 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	plan := route.Plan{
		ModuleMoveID: "move-1",
		Segments: []route.Segment{
			{From: "yard-a", To: "staging-b", Lift: "lift-1", Depart: now, Arrive: now.Add(time.Hour)},
			{From: "staging-b", To: "site-c", Lift: "lift-2", Depart: now.Add(2 * time.Hour), Arrive: now.Add(3 * time.Hour)},
		},
		TotalMinutes: 180,
		TotalStops:   1,
	}
	missingVehicle := Assignment{ID: "assignment-missing", ModuleMoveID: "move-1", VehicleID: "vehicle-missing", WeightKg: 12}
	if err := planner.AssignPlanned(context.Background(), missingVehicle, plan, now); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing vehicle error = %v, want not found", err)
	}

	assignment := Assignment{ID: "assignment-1", ModuleMoveID: "move-1", VehicleID: "vehicle-1", WeightKg: 12}
	if err := planner.AssignPlanned(context.Background(), assignment, plan, now); err != nil {
		t.Fatalf("assign planned route: %v", err)
	}
	plan.Segments[0].To = "preview-only-detour"
	plan.Segments[0].Lift = "preview-lift"
	plan.Segments = append(plan.Segments, route.Segment{From: "site-c", To: "holding-d", Lift: "lift-3"})

	stored, err := planner.Describe("assignment-1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "assigned" || stored.VehicleID != "vehicle-1" || stored.ModuleMoveID != "move-1" || stored.WeightKg != 12 {
		t.Errorf("stored assignment metadata changed: %+v", stored)
	}
	if len(stored.Segments) != 2 {
		t.Errorf("stored route has %d segments, want 2", len(stored.Segments))
	} else if stored.Segments[0].To != "staging-b" || stored.Segments[0].Lift != "lift-1" {
		t.Errorf("editing the route preview changed the stored assignment: %+v", stored.Segments[0])
	}
}
