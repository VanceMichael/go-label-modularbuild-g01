package transport

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/route"
)

// TestPlannedPreviewIsolation reproduces the field-process scenario:
// vehicle-1, SITE-LIFT-01, move-1, yard-a, staging-b, lift-1, site-c, lift-2,
// assignment-missing, vehicle-missing, assignment-1, preview-only-detour,
// preview-lift, holding-d, lift-3.
func TestPlannedPreviewIsolation(t *testing.T) {
	p := New()
	now := time.Now()
	if err := p.AddVehicle(Vehicle{
		ID:            "vehicle-1",
		Registration:  "SITE-LIFT-01",
		Carrier:       "move-1",
		MaxKg:         100,
		AvailableFrom: now.Add(-time.Hour),
		AvailableTo:   now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	// Assignment against a missing vehicle must surface ErrNotFound, not a
	// wrapped/opaque error.
	if err := p.Assign(context.Background(), Assignment{
		ID: "assignment-missing", ModuleMoveID: "move-1",
		VehicleID: "vehicle-missing", WeightKg: 10,
	}, now); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing vehicle: want ErrNotFound, got %v", err)
	}

	// A planned assignment for a legitimate vehicle keeps its vehicle,
	// module and weight fields and is marked assigned.
	preview := route.Plan{
		ModuleMoveID: "move-1",
		Segments: []route.Segment{
			{From: "yard-a", To: "staging-b", Lift: "lift-1", AvailableKg: 100,
				Depart: now.Add(time.Minute), Arrive: now.Add(2 * time.Minute)},
			{From: "staging-b", To: "site-c", Lift: "lift-2", AvailableKg: 100,
				Depart: now.Add(3 * time.Minute), Arrive: now.Add(4 * time.Minute)},
		},
	}
	got := Assignment{
		ID: "assignment-1", ModuleMoveID: "move-1",
		VehicleID: "vehicle-1", WeightKg: 10,
	}
	if err := p.AssignPlanned(context.Background(), got, preview, now); err != nil {
		t.Fatalf("AssignPlanned: %v", err)
	}
	stored, err := p.Describe("assignment-1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "assigned" {
		t.Fatalf("status: want assigned, got %s", stored.Status)
	}
	if stored.VehicleID != "vehicle-1" || stored.ModuleMoveID != "move-1" || stored.WeightKg != 10 {
		t.Fatalf("fields not retained: %#v", stored)
	}

	// The caller now rewrites / extends the preview slice in place to model a
	// preview-only detour: first segment destination becomes the detour
	// "preview-only-detour" with lift "preview-lift", and an extra tail segment
	// through "holding-d" with lift "lift-3" is appended.
	preview.Segments[0].To = "preview-only-detour"
	preview.Segments[0].Lift = "preview-lift"
	preview.Segments = append(preview.Segments, route.Segment{
		From: "site-c", To: "holding-d", Lift: "lift-3", AvailableKg: 100,
		Depart: now.Add(5 * time.Minute), Arrive: now.Add(6 * time.Minute),
	})

	// The stored assignment must keep its original two segments: first segment
	// destination staging-b, lift resource lift-1.
	stored2, err := p.Describe("assignment-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(stored2.Segments) != 2 {
		t.Fatalf("segment count: want 2, got %d", len(stored2.Segments))
	}
	if stored2.Segments[0].To != "staging-b" {
		t.Fatalf("first segment destination: want staging-b, got %s", stored2.Segments[0].To)
	}
	if stored2.Segments[0].Lift != "lift-1" {
		t.Fatalf("first segment lift: want lift-1, got %s", stored2.Segments[0].Lift)
	}
}
