package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"sort"
	"time"
)

type Snapshot struct {
	Version     int64
	TenantID    string
	CreatedAt   time.Time
	ModuleMoves []domain.ModuleMove
	Legs        []domain.LiftWindow
	Hash        string
}

func Make(version int64, tenant string, module_moves []domain.ModuleMove, windows []domain.LiftWindow, at time.Time) (Snapshot, error) {
	if version < 1 || tenant == "" || at.IsZero() {
		return Snapshot{}, domain.ErrInvalid
	}
	s := Snapshot{Version: version, TenantID: tenant, CreatedAt: at, ModuleMoves: append([]domain.ModuleMove(nil), module_moves...), Legs: append([]domain.LiftWindow(nil), windows...)}
	sort.Slice(s.ModuleMoves, func(i, j int) bool { return s.ModuleMoves[i].ID < s.ModuleMoves[j].ID })
	sort.Slice(s.Legs, func(i, j int) bool { return s.Legs[i].ID < s.Legs[j].ID })
	s.Hash = hash(s)
	return s, nil
}
func (s Snapshot) EqualPayload(other Snapshot) bool {
	return s.Hash == other.Hash && s.TenantID == other.TenantID && s.Version == other.Version
}
func (s Snapshot) Validate() error {
	if s.Version < 1 || s.TenantID == "" || s.Hash == "" {
		return domain.ErrInvalid
	}
	if s.Hash != hash(s) {
		return domain.ErrConflict
	}
	return nil
}
func hash(s Snapshot) string {
	b, _ := json.Marshal(struct {
		V int64
		T string
		S []domain.ModuleMove
		L []domain.LiftWindow
	}{s.Version, s.TenantID, s.ModuleMoves, s.Legs})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
