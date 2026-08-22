package windpolicy

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type tenantEntry struct {
	mu sync.Mutex

	done   bool
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

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.done {
		if entry.err != nil {
			return Policy{}, fmt.Errorf("load active wind policy: %w", entry.err)
		}
		return entry.policy, nil
	}

	// Honor an already-cancelled context before starting the loader.
	if err := ctx.Err(); err != nil {
		return Policy{}, err
	}

	policy, err := c.loader.LoadActive(ctx, tenantID)
	if err == nil && policy.TenantID != tenantID {
		policy = Policy{}
		err = fmt.Errorf("wind policy: loader returned another tenant")
	}
	// A cancellation (or deadline) observed while the loader was running is
	// transient: surface it to the caller without caching the failure so the
	// next call can retry the load from scratch.
	if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		return Policy{}, err
	}

	entry.policy = policy
	entry.err = err
	entry.done = true

	if entry.err != nil {
		return Policy{}, fmt.Errorf("load active wind policy: %w", entry.err)
	}
	return entry.policy, nil
}
