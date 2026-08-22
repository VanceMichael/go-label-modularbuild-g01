package aggregate

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"testing"
)

func TestAggregateVersion(t *testing.T) {
	s := New()
	if err := s.Create(ModuleMove{ID: "S", Status: domain.ModuleMoveDraft, Version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition("S", domain.ModuleMoveBooked, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition("S", domain.ModuleMoveScreening, 1); err != domain.ErrConflict {
		t.Fatal(err)
	}
	v, _ := s.Get("S")
	if v.Version != 2 || len(v.Events) != 1 {
		t.Fatal(v)
	}
}
