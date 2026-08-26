package windpolicy

import (
	"context"
	"fmt"
	"sync"
)

type tenantEntry struct {
	once   sync.Once
	policy Policy
	err    error
}

type Cache struct {
	loader Loader

	mu      sync.Mutex
	entries map[string]*tenantEntry
}

func NewCache(loader Loader) *Cache {
	return &Cache{loader: loader, entries: make(map[string]*tenantEntry)}
}

func (c *Cache) Active(ctx context.Context, tenantID string) (Policy, error) {
	if tenantID == "" {
		return Policy{}, fmt.Errorf("wind policy: tenant is required")
	}
	c.mu.Lock()
	entry := c.entries[tenantID]
	if entry == nil {
		entry = &tenantEntry{}
		c.entries[tenantID] = entry
	}
	c.mu.Unlock()

	entry.once.Do(func() {
		entry.policy, entry.err = c.loader.LoadActive(ctx, tenantID)
		if entry.err == nil && entry.policy.TenantID != tenantID {
			entry.policy = Policy{}
			entry.err = fmt.Errorf("wind policy: loader returned another tenant")
		}
	})
	if entry.err != nil {
		return Policy{}, fmt.Errorf("load active wind policy: %w", entry.err)
	}
	return entry.policy, nil
}
