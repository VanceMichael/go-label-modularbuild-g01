package finance

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"testing"
	"time"
)

func TestLedgerRejectsInvalid(t *testing.T) {
	l := New()
	if err := l.Post(Entry{ID: "", TenantID: "T"}); err != domain.ErrInvalid {
		t.Fatal(err)
	}
	now := time.Now()
	if err := l.Post(Entry{ID: "x", TenantID: "T", ModuleMoveID: "S", Currency: "CNY", Debit: 1, Credit: 1, PostedAt: now}); err != domain.ErrInvalid {
		t.Fatal(err)
	}
	if err := l.Post(Entry{ID: "x", TenantID: "T", ModuleMoveID: "S", Currency: "CNY", Credit: 1, PostedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := l.Post(Entry{ID: "x", TenantID: "T", ModuleMoveID: "S", Currency: "CNY", Credit: 1, PostedAt: now}); err != domain.ErrConflict {
		t.Fatal(err)
	}
}
