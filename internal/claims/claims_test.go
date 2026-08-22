package claims

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"testing"
	"time"
)

func TestClaimState(t *testing.T) {
	now := time.Now()
	c := Claim{ID: "C", TenantID: "T", ModuleMoveID: "S", FiledBy: "u", Reason: "late", FiledAt: now, Status: "open"}
	if err := Open(c); err != nil {
		t.Fatal(err)
	}
	c, err := Resolve(c, now.Add(time.Minute))
	if err != nil || c.Status != "resolved" {
		t.Fatal(err)
	}
	if _, err := Resolve(c, now.Add(time.Hour)); err != domain.ErrState {
		t.Fatal(err)
	}
}
