package report

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"testing"
	"time"
)

func TestBuildSummary(t *testing.T) {
	now := time.Now()
	s := Build([]ModuleMoveRow{{Reference: "A", Status: domain.ModuleMoveBooked, WeightKg: 10, CreatedAt: now}, {Reference: "B", Status: domain.ModuleMoveDraft, WeightKg: 5, CreatedAt: now.Add(time.Minute)}}, 1)
	if s.Total != 2 || s.WeightKg != 15 || len(s.Latest) != 1 {
		t.Fatalf("%#v", s)
	}
	if s.ByStatus[domain.ModuleMoveDraft] != 1 {
		t.Fatal(s.ByStatus)
	}
}
