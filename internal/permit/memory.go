package permit

import (
	"context"
	"sync"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type MemoryStore struct {
	mu            sync.Mutex
	permits       map[string]Permit
	audits        []AuditEvent
	failNextAudit bool
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{permits: make(map[string]Permit)}
}

func permitKey(tenantID, permitID string) string {
	return tenantID + "|" + permitID
}

func (m *MemoryStore) Add(permit Permit) error {
	if permit.ID == "" || permit.TenantID == "" || permit.ModuleID == "" {
		return domain.ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := permitKey(permit.TenantID, permit.ID)
	if _, exists := m.permits[key]; exists {
		return domain.ErrConflict
	}
	permit.Status = StatusPending
	permit.Version = 1
	m.permits[key] = permit
	return nil
}

func (m *MemoryStore) FailNextAudit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNextAudit = true
}

func (m *MemoryStore) Transaction(ctx context.Context, fn func(Tx) error) error {
	if fn == nil {
		return domain.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	permits := make(map[string]Permit, len(m.permits))
	for key, value := range m.permits {
		permits[key] = value
	}
	audits := append([]AuditEvent(nil), m.audits...)
	tx := &memoryTx{permits: permits, audits: audits, failAudit: m.failNextAudit}
	m.failNextAudit = false
	if err := fn(tx); err != nil {
		return err
	}
	m.permits = tx.permits
	m.audits = tx.audits
	return nil
}

func (m *MemoryStore) GetPermit(ctx context.Context, tenantID, permitID string) (Permit, error) {
	if err := ctx.Err(); err != nil {
		return Permit{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	permit, exists := m.permits[permitKey(tenantID, permitID)]
	if !exists {
		return Permit{}, domain.ErrNotFound
	}
	return permit, nil
}

func (m *MemoryStore) AuditCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.audits)
}

type memoryTx struct {
	permits   map[string]Permit
	audits    []AuditEvent
	failAudit bool
}

func (tx *memoryTx) GetPermit(ctx context.Context, tenantID, permitID string) (Permit, error) {
	if err := ctx.Err(); err != nil {
		return Permit{}, err
	}
	permit, exists := tx.permits[permitKey(tenantID, permitID)]
	if !exists {
		return Permit{}, domain.ErrNotFound
	}
	return permit, nil
}

func (tx *memoryTx) SavePermit(ctx context.Context, permit Permit) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := permitKey(permit.TenantID, permit.ID)
	if _, exists := tx.permits[key]; !exists {
		return domain.ErrNotFound
	}
	tx.permits[key] = permit
	return nil
}

func (tx *memoryTx) AppendAudit(ctx context.Context, event AuditEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if tx.failAudit {
		tx.failAudit = false
		return ErrAuditUnavailable
	}
	tx.audits = append(tx.audits, event)
	return nil
}

type MemoryCache struct {
	mu      sync.Mutex
	permits map[string]Permit
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{permits: make(map[string]Permit)}
}

func (c *MemoryCache) Put(ctx context.Context, permit Permit) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.permits[permitKey(permit.TenantID, permit.ID)] = permit
	return nil
}

func (c *MemoryCache) Get(ctx context.Context, tenantID, permitID string) (Permit, error) {
	if err := ctx.Err(); err != nil {
		return Permit{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	permit, exists := c.permits[permitKey(tenantID, permitID)]
	if !exists {
		return Permit{}, domain.ErrNotFound
	}
	return permit, nil
}
