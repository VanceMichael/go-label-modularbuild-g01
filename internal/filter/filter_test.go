package filter

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"testing"
	"time"
)

func TestApplyFilters(t *testing.T) {
	now := time.Now()
	items := []ModuleMove{{ID: "1", TenantID: "A", Reference: "ABC", Status: domain.ModuleMoveBooked, Origin: "PEK", Destination: "FRA", WeightKg: 20, CreatedAt: now}, {ID: "2", TenantID: "B", Reference: "ABC", Status: domain.ModuleMoveBooked, Origin: "PEK", Destination: "FRA", WeightKg: 20, CreatedAt: now}}
	out := Apply(items, Input{TenantID: "A", Origin: "PEK", MaxWeight: 30})
	if len(out) != 1 || out[0].ID != "1" {
		t.Fatalf("%#v", out)
	}
	out = Apply(items, Input{Statuses: []domain.ModuleMoveStatus{domain.ModuleMoveDraft}})
	if len(out) != 0 {
		t.Fatal(out)
	}
}
