package safetyalarm

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type subscription struct {
	id       string
	tenantID string
	handler  Handler
}

type Registry struct {
	mu            sync.Mutex
	subscriptions map[string]subscription
}

func NewRegistry() *Registry {
	return &Registry{subscriptions: make(map[string]subscription)}
}

func (r *Registry) Subscribe(tenantID, subscriptionID string, handler Handler) error {
	if tenantID == "" || subscriptionID == "" || handler == nil {
		return domain.ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.subscriptions[subscriptionID]; exists {
		return domain.ErrConflict
	}
	r.subscriptions[subscriptionID] = subscription{id: subscriptionID, tenantID: tenantID, handler: handler}
	return nil
}

func (r *Registry) Unsubscribe(tenantID, subscriptionID string) error {
	if tenantID == "" || subscriptionID == "" {
		return domain.ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	subscription, exists := r.subscriptions[subscriptionID]
	if !exists || subscription.tenantID != tenantID {
		return domain.ErrNotFound
	}
	delete(r.subscriptions, subscriptionID)
	return nil
}

func (r *Registry) Dispatch(ctx context.Context, alarm Alarm) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0)
	for id, subscription := range r.subscriptions {
		if subscription.tenantID == alarm.TenantID {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := r.subscriptions[id].handler.Handle(ctx, alarm); err != nil {
			return fmt.Errorf("deliver safety alarm to %s: %w", id, err)
		}
	}
	return nil
}

func (r *Registry) Count(tenantID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, subscription := range r.subscriptions {
		if subscription.tenantID == tenantID {
			count++
		}
	}
	return count
}
