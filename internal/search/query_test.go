package search

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"testing"
)

func TestFilter(t *testing.T) {
	items := []Item{{ID: "2", Reference: "ABC-2", Status: domain.ModuleMoveBooked}, {ID: "1", Reference: "ABC-1", Status: domain.ModuleMoveDraft}}
	out := Filter(items, Query{Term: "ABC", Statuses: []domain.ModuleMoveStatus{domain.ModuleMoveBooked}})
	if len(out) != 1 || out[0].ID != "2" {
		t.Fatalf("%#v", out)
	}
}
