package scan

import (
	"context"
	"fmt"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"sync"
	"time"
)

type Result struct {
	ModuleMoveID string
	Status       domain.SiteSafetyStatus
	Officer      string
	Reason       string
	CheckedAt    time.Time
}
type Service struct {
	mu      sync.RWMutex
	results map[string]Result
	blocked map[string]string
}

func New() *Service { return &Service{results: map[string]Result{}, blocked: map[string]string{}} }
func (s *Service) Check(ctx context.Context, module_move, officer string, signals map[string]bool) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	default:
	}
	if module_move == "" || officer == "" {
		return Result{}, domain.ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, bad := range signals {
		if bad {
			r := Result{ModuleMoveID: module_move, Officer: officer, Status: domain.SiteSafetyFailed, Reason: fmt.Sprintf("signal %s", name), CheckedAt: time.Now().UTC()}
			s.results[module_move] = r
			s.blocked[module_move] = r.Reason
			return r, nil
		}
	}
	r := Result{ModuleMoveID: module_move, Officer: officer, Status: domain.SiteSafetyPassed, CheckedAt: time.Now().UTC()}
	s.results[module_move] = r
	delete(s.blocked, module_move)
	return r, nil
}
func (s *Service) Get(module_move string) (Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.results[module_move]
	if !ok {
		return Result{}, domain.ErrNotFound
	}
	return r, nil
}
func (s *Service) Blocked(module_move string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.blocked[module_move]
	return ok
}
