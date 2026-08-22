package search

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"testing"
)

func TestSearchOrdering(t *testing.T) {
	items := []Item{{ID: "2", Reference: "ABC-2"}, {ID: "1", Reference: "ABC"}, {ID: "3", Reference: "XYZ"}}
	out := Filter(items, Query{Term: "ABC", Limit: 2})
	if len(out) != 2 || out[0].ID != "1" {
		t.Fatalf("%#v", out)
	}
	out = Filter(items, Query{Statuses: []domain.ModuleMoveStatus{domain.ModuleMoveDeparted}})
	if len(out) != 0 {
		t.Fatal(out)
	}
}
