package liftmethod

import (
	"context"
	"sync"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Memory struct {
	mu         sync.Mutex
	statements map[string]Statement
	audits     []Statement
}

func NewMemory() *Memory {
	return &Memory{statements: make(map[string]Statement)}
}

func statementKey(tenantID, statementID string) string {
	return tenantID + "|" + statementID
}

func (m *Memory) Add(statement Statement) error {
	if statement.ID == "" || statement.TenantID == "" || statement.Title == "" || statement.Revision == "" {
		return domain.ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := statementKey(statement.TenantID, statement.ID)
	if _, exists := m.statements[key]; exists {
		return domain.ErrConflict
	}
	statement.Version = 1
	m.statements[key] = statement
	return nil
}

func (m *Memory) Get(ctx context.Context, tenantID, statementID string) (Statement, error) {
	if err := ctx.Err(); err != nil {
		return Statement{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	statement, exists := m.statements[statementKey(tenantID, statementID)]
	if !exists {
		return Statement{}, domain.ErrNotFound
	}
	return statement, nil
}

func (m *Memory) Save(ctx context.Context, statement Statement, expectedVersion int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := statementKey(statement.TenantID, statement.ID)
	if _, exists := m.statements[key]; !exists {
		return domain.ErrNotFound
	}
	statement.Version = expectedVersion + 1
	m.statements[key] = statement
	m.audits = append(m.audits, statement)
	return nil
}

func (m *Memory) AuditCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.audits)
}
