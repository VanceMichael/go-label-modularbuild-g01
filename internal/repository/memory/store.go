package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"sort"
	"sync"
	"time"
)

type Store struct {
	mu           sync.RWMutex
	users        map[string]domain.User
	byEmail      map[string]string
	sessions     map[string]domain.Session
	module_moves map[string]domain.ModuleMove
	idem         map[string]string
	windows      map[string]domain.LiftWindow
	quality      map[string]domain.QualityCase
	site_safety  map[string]domain.SiteSafetyCheck
	audit        []domain.AuditEvent
	outbox       map[string]domain.OutboxEvent
}

func New() *Store {
	return &Store{users: map[string]domain.User{}, byEmail: map[string]string{}, sessions: map[string]domain.Session{}, module_moves: map[string]domain.ModuleMove{}, idem: map[string]string{}, windows: map[string]domain.LiftWindow{}, quality: map[string]domain.QualityCase{}, site_safety: map[string]domain.SiteSafetyCheck{}, outbox: map[string]domain.OutboxEvent{}}
}
func (s *Store) Ping(context.Context) error { return nil }
func (s *Store) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return cloneUser(s.users[id]), nil
}
func (s *Store) GetUser(_ context.Context, id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return cloneUser(u), nil
}
func (s *Store) CreateUser(_ context.Context, u domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byEmail[u.Email]; exists {
		return domain.ErrConflict
	}
	s.users[u.ID] = cloneUser(u)
	s.byEmail[u.Email] = u.ID
	return nil
}
func (s *Store) DeactivateUser(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	u.Active = false
	u.DeactivatedAt = &at
	s.users[id] = u
	for key, session := range s.sessions {
		if session.UserID == id {
			session.RevokedAt = &at
			s.sessions[key] = session
		}
	}
	return nil
}
func (s *Store) CreateSession(_ context.Context, v domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[v.TokenHash] = v
	return nil
}
func (s *Store) GetSession(_ context.Context, token string) (domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.sessions[tokenHash(token)]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	return v, nil
}
func (s *Store) RevokeSession(_ context.Context, token string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tokenHash(token)
	v, ok := s.sessions[key]
	if !ok {
		return domain.ErrNotFound
	}
	v.RevokedAt = &at
	s.sessions[key] = v
	return nil
}
func (s *Store) RevokeUserSessions(_ context.Context, user string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, v := range s.sessions {
		if v.UserID == user {
			v.RevokedAt = &at
			s.sessions[key] = v
		}
	}
	return nil
}
func (s *Store) CreateModuleMove(_ context.Context, v domain.ModuleMove) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := v.TenantID + "|" + v.IdempotencyKey
	if v.IdempotencyKey != "" {
		if old, ok := s.idem[key]; ok {
			if old == v.ID {
				return nil
			}
			return domain.ErrConflict
		}
	}
	if _, ok := s.module_moves[v.ID]; ok {
		return domain.ErrConflict
	}
	s.module_moves[v.ID] = v
	if v.IdempotencyKey != "" {
		s.idem[key] = v.ID
	}
	return nil
}
func (s *Store) GetModuleMove(_ context.Context, tenant, id string) (domain.ModuleMove, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.module_moves[id]
	if !ok || v.TenantID != tenant {
		return domain.ModuleMove{}, domain.ErrNotFound
	}
	return v, nil
}
func (s *Store) UpdateModuleMove(_ context.Context, v domain.ModuleMove, version int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.module_moves[v.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if old.Version != version {
		return domain.ErrConflict
	}
	v.Version = version + 1
	s.module_moves[v.ID] = v
	return nil
}
func (s *Store) ListModuleMoves(_ context.Context, tenant string, req domain.PageRequest) (domain.Page[domain.ModuleMove], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req = req.Normalized()
	all := make([]domain.ModuleMove, 0)
	for _, v := range s.module_moves {
		if v.TenantID == tenant {
			all = append(all, v)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.Before(all[j].CreatedAt) })
	total := len(all)
	if len(all) > req.Limit {
		all = all[:req.Limit]
	}
	return domain.Page[domain.ModuleMove]{Items: all, Total: total}, nil
}
func (s *Store) FindByIdempotency(_ context.Context, tenant, key string) (domain.ModuleMove, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.idem[tenant+"|"+key]
	if !ok {
		return domain.ModuleMove{}, domain.ErrNotFound
	}
	return s.module_moves[id], nil
}
func (s *Store) CreateLeg(_ context.Context, v domain.LiftWindow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.windows[v.ID]; ok {
		return domain.ErrConflict
	}
	s.windows[v.ID] = v
	return nil
}
func (s *Store) GetLeg(_ context.Context, tenant, id string) (domain.LiftWindow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.windows[id]
	if !ok || v.TenantID != tenant {
		return domain.LiftWindow{}, domain.ErrNotFound
	}
	return v, nil
}
func (s *Store) ReserveCapacity(_ context.Context, tenant, id string, weight, version int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.windows[id]
	if !ok || v.TenantID != tenant {
		return domain.ErrNotFound
	}
	if v.Version != version {
		return domain.ErrConflict
	}
	if err := domain.ValidateCapacity(v, weight); err != nil {
		return err
	}
	v.ReservedKg += weight
	v.Version++
	s.windows[id] = v
	return nil
}
func (s *Store) UpdateWindowStatus(_ context.Context, tenant, id string, status domain.WindowStatus, version int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.windows[id]
	if !ok || v.TenantID != tenant {
		return domain.ErrNotFound
	}
	if v.Version != version {
		return domain.ErrConflict
	}
	if !v.Status.CanTransition(status) {
		return domain.ErrState
	}
	v.Status = status
	v.Version++
	s.windows[id] = v
	return nil
}
func (s *Store) BookModuleMove(_ context.Context, tenant, module_moveID, windowID string, module_moveVersion, windowVersion int64, at time.Time) (domain.ModuleMove, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	module_move, ok := s.module_moves[module_moveID]
	if !ok || module_move.TenantID != tenant {
		return domain.ModuleMove{}, domain.ErrNotFound
	}
	window, ok := s.windows[windowID]
	if !ok || window.TenantID != tenant {
		return domain.ModuleMove{}, domain.ErrNotFound
	}
	if module_move.Version != module_moveVersion || window.Version != windowVersion {
		return domain.ModuleMove{}, domain.ErrConflict
	}
	if !module_move.CanTransition(domain.ModuleMoveBooked) || window.Status != domain.WindowOpen {
		return domain.ModuleMove{}, domain.ErrState
	}
	if module_move.Origin != window.Origin || module_move.Destination != window.Destination {
		return domain.ModuleMove{}, domain.ErrInvalid
	}
	if err := domain.ValidateCapacity(window, module_move.WeightKg); err != nil {
		return domain.ModuleMove{}, err
	}
	window.ReservedKg += module_move.WeightKg
	window.Version++
	module_move.Status = domain.ModuleMoveBooked
	module_move.LegID = &windowID
	module_move.UpdatedAt = at
	module_move.Version++
	s.windows[windowID] = window
	s.module_moves[module_moveID] = module_move
	return module_move, nil
}
func (s *Store) GetQuality(_ context.Context, id string) (domain.QualityCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.quality[id]
	if !ok {
		return domain.QualityCase{}, domain.ErrNotFound
	}
	return v, nil
}
func (s *Store) PutQuality(_ context.Context, v domain.QualityCase) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quality[v.ID] = v
	return nil
}
func (s *Store) GetSiteSafety(_ context.Context, id string) (domain.SiteSafetyCheck, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.site_safety[id]
	if !ok {
		return domain.SiteSafetyCheck{}, domain.ErrNotFound
	}
	return v, nil
}
func (s *Store) PutSiteSafety(_ context.Context, v domain.SiteSafetyCheck) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.site_safety[v.ID] = v
	return nil
}
func (s *Store) AppendAudit(_ context.Context, v domain.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audit = append(s.audit, v)
	return nil
}
func (s *Store) ListAudit(_ context.Context, tenant string, req domain.PageRequest) (domain.Page[domain.AuditEvent], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.AuditEvent, 0)
	for _, v := range s.audit {
		if v.TenantID == tenant {
			out = append(out, v)
		}
	}
	return domain.Page[domain.AuditEvent]{Items: out, Total: len(out)}, nil
}
func (s *Store) Enqueue(_ context.Context, v domain.OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outbox[v.ID] = v
	return nil
}
func (s *Store) Claim(_ context.Context, now time.Time, n int) ([]domain.OutboxEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.OutboxEvent, 0)
	for id, v := range s.outbox {
		if v.PublishedAt == nil && v.ClaimedAt == nil && !v.AvailableAt.After(now) && len(out) < n {
			v.ClaimedAt = &now
			v.Attempts++
			s.outbox[id] = v
			out = append(out, v)
		}
	}
	return out, nil
}
func (s *Store) MarkPublished(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.outbox[id]
	if !ok {
		return domain.ErrNotFound
	}
	v.PublishedAt = &at
	s.outbox[id] = v
	return nil
}
func (s *Store) MarkFailed(_ context.Context, id string, at time.Time, err string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.outbox[id]
	if !ok {
		return domain.ErrNotFound
	}
	v.LastError = err
	v.ClaimedAt = nil
	v.AvailableAt = at.Add(time.Minute * time.Duration(v.Attempts))
	s.outbox[id] = v
	return nil
}
func (s *Store) Summary(_ context.Context, tenant string) (int, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pending, failed := 0, 0
	for _, v := range s.outbox {
		if v.TenantID == tenant && v.PublishedAt == nil {
			pending++
			if v.Attempts >= 5 {
				failed++
			}
		}
	}
	return pending, failed, nil
}
func tokenHash(v string) string { sum := sha256.Sum256([]byte(v)); return hex.EncodeToString(sum[:]) }
func cloneUser(v domain.User) domain.User {
	v.PasswordHash = append([]byte(nil), v.PasswordHash...)
	return v
}
