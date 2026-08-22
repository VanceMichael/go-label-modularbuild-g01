package handover

import (
	"fmt"
	"sync"
)

type MemoryStore struct {
	mu      sync.Mutex
	batches map[string]Batch
	modules map[string]Module
	audits  []AuditEntry
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		batches: make(map[string]Batch),
		modules: make(map[string]Module),
	}
}

func (s *MemoryStore) Seed(batch Batch, modules ...Module) {
	s.mu.Lock()
	defer s.mu.Unlock()
	batch.ModuleIDs = append([]string(nil), batch.ModuleIDs...)
	s.batches[batch.ID] = batch
	for _, module := range modules {
		s.modules[module.ID] = module
	}
}

func (s *MemoryStore) Snapshot(batchID string) (Batch, []Module, []AuditEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	batch, ok := s.batches[batchID]
	if !ok {
		return Batch{}, nil, nil, false
	}
	batch.ModuleIDs = append([]string(nil), batch.ModuleIDs...)
	modules := make([]Module, 0, len(batch.ModuleIDs))
	for _, id := range batch.ModuleIDs {
		modules = append(modules, s.modules[id])
	}
	return batch, modules, append([]AuditEntry(nil), s.audits...), true
}

func (s *MemoryStore) Begin(batchID string) (*Tx, error) {
	s.mu.Lock()
	batch, ok := s.batches[batchID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("batch %s not found", batchID)
	}
	modules := make(map[string]Module, len(batch.ModuleIDs))
	for _, id := range batch.ModuleIDs {
		module, exists := s.modules[id]
		if !exists {
			s.mu.Unlock()
			return nil, fmt.Errorf("module %s not found", id)
		}
		modules[id] = module
	}
	batch.ModuleIDs = append([]string(nil), batch.ModuleIDs...)
	return &Tx{store: s, batch: batch, modules: modules}, nil
}

type Tx struct {
	store   *MemoryStore
	batch   Batch
	modules map[string]Module
	audits  []AuditEntry
	closed  bool
}

func (tx *Tx) Batch() Batch { return tx.batch }

func (tx *Tx) Module(id string) (Module, bool) {
	module, ok := tx.modules[id]
	return module, ok
}

func (tx *Tx) MarkModule(module Module, audit AuditEntry) {
	tx.modules[module.ID] = module
	tx.audits = append(tx.audits, audit)
}

func (tx *Tx) CompleteBatch(batch Batch) { tx.batch = batch }

func (tx *Tx) Commit() error {
	if tx.closed {
		return errorsNewClosedTransaction()
	}
	defer tx.store.mu.Unlock()
	for id, module := range tx.modules {
		tx.store.modules[id] = module
	}
	tx.store.batches[tx.batch.ID] = tx.batch
	tx.store.audits = append(tx.store.audits, tx.audits...)
	tx.closed = true
	return nil
}

func (tx *Tx) Rollback() {
	if tx.closed {
		return
	}
	tx.closed = true
	tx.store.mu.Unlock()
}

func errorsNewClosedTransaction() error {
	return fmt.Errorf("handover: transaction already closed")
}
