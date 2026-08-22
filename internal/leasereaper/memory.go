package leasereaper

import (
	"context"
	"sync"
	"time"
)

type Memory struct {
	mu     sync.Mutex
	leases map[string]Lease
}

func NewMemory(initial ...Lease) *Memory {
	m := &Memory{leases: make(map[string]Lease)}
	for _, lease := range initial {
		m.leases[key(lease.TenantID, lease.Resource)] = lease
	}
	return m
}

func (m *Memory) Expired(ctx context.Context, at time.Time) ([]Lease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Lease, 0)
	for _, lease := range m.leases {
		if lease.Expires.Before(at) {
			result = append(result, lease)
		}
	}
	return result, nil
}

func (m *Memory) Delete(ctx context.Context, tenantID, resource string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.leases, key(tenantID, resource))
	return nil
}

func (m *Memory) Renew(ctx context.Context, tenantID, resource string, expires time.Time) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	lease, ok := m.leases[key(tenantID, resource)]
	if !ok {
		return Lease{}, ErrLeaseMissing
	}
	lease.Version++
	lease.Expires = expires
	m.leases[key(tenantID, resource)] = lease
	return lease, nil
}

func (m *Memory) Get(tenantID, resource string) (Lease, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lease, ok := m.leases[key(tenantID, resource)]
	return lease, ok
}

func key(tenantID, resource string) string { return tenantID + "/" + resource }
