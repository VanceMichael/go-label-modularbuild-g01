package quality

import (
	"context"
	"fmt"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"strings"
	"sync"
	"time"
)

type Document struct {
	Number    string
	Kind      string
	IssuedAt  time.Time
	ExpiresAt time.Time
	Hash      string
}
type Case struct {
	ID           string
	ModuleMoveID string
	TenantID     string
	Documents    []Document
	Status       domain.QualityStatus
	Notes        []string
	UpdatedAt    time.Time
}
type Workflow struct {
	mu    sync.RWMutex
	cases map[string]Case
}

func NewWorkflow() *Workflow { return &Workflow{cases: map[string]Case{}} }
func (w *Workflow) Open(_ context.Context, c Case) error {
	if c.ID == "" || c.ModuleMoveID == "" || c.TenantID == "" {
		return domain.ErrInvalid
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.cases[c.ModuleMoveID]; ok {
		return domain.ErrConflict
	}
	c.Status = domain.QualityPending
	c.UpdatedAt = time.Now().UTC()
	w.cases[c.ModuleMoveID] = c
	return nil
}
func (w *Workflow) Attach(_ context.Context, module_move string, d Document) error {
	if strings.TrimSpace(d.Number) == "" || d.ExpiresAt.Before(d.IssuedAt) {
		return domain.ErrInvalid
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	c, ok := w.cases[module_move]
	if !ok {
		return domain.ErrNotFound
	}
	for _, old := range c.Documents {
		if old.Number == d.Number {
			return domain.ErrConflict
		}
	}
	c.Documents = append(c.Documents, d)
	c.UpdatedAt = time.Now().UTC()
	w.cases[module_move] = c
	return nil
}
func (w *Workflow) AttachBatch(ctx context.Context, moduleMove string, documents []Document) error {
	if len(documents) == 0 {
		return domain.ErrInvalid
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	c, ok := w.cases[moduleMove]
	if !ok {
		return domain.ErrNotFound
	}
	// Validate the whole batch before mutating so a later invalid document
	// leaves the case's existing documents untouched.
	seen := make(map[string]struct{}, len(c.Documents)+len(documents))
	for _, attached := range c.Documents {
		seen[attached.Number] = struct{}{}
	}
	for _, document := range documents {
		if strings.TrimSpace(document.Number) == "" || document.ExpiresAt.Before(document.IssuedAt) {
			return domain.ErrInvalid
		}
		if _, dup := seen[document.Number]; dup {
			return domain.ErrConflict
		}
		seen[document.Number] = struct{}{}
	}
	c.Documents = append(c.Documents, documents...)
	c.UpdatedAt = time.Now().UTC()
	w.cases[moduleMove] = c
	return nil
}
func (w *Workflow) Review(_ context.Context, module_move, actor string, now time.Time) (Case, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	c, ok := w.cases[module_move]
	if !ok {
		return Case{}, domain.ErrNotFound
	}
	if actor == "" {
		return Case{}, domain.ErrForbidden
	}
	if len(c.Documents) == 0 {
		return Case{}, fmt.Errorf("%w: documents missing", domain.ErrState)
	}
	for _, d := range c.Documents {
		if !now.Before(d.ExpiresAt) {
			c.Status = domain.QualityHeld
			c.Notes = append(c.Notes, "expired document "+d.Number)
			w.cases[module_move] = c
			return c, domain.ErrExpired
		}
	}
	c.Status = domain.QualityReleased
	c.Notes = append(c.Notes, "reviewed by "+actor)
	c.UpdatedAt = now
	w.cases[module_move] = c
	return c, nil
}
func (w *Workflow) Get(module_move string) (Case, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	c, ok := w.cases[module_move]
	if !ok {
		return Case{}, domain.ErrNotFound
	}
	c.Documents = append([]Document(nil), c.Documents...)
	c.Notes = append([]string(nil), c.Notes...)
	return c, nil
}
