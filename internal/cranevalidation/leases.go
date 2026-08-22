package cranevalidation

import "sync"

type Leases struct {
	mu   sync.Mutex
	held map[string]bool
}

func NewLeases() *Leases { return &Leases{held: map[string]bool{}} }
func (l *Leases) Acquire(key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.held[key] {
		return ErrLeaseHeld
	}
	l.held[key] = true
	return nil
}
func (l *Leases) Release(key string)   { l.mu.Lock(); defer l.mu.Unlock(); delete(l.held, key) }
func (l *Leases) Held(key string) bool { l.mu.Lock(); defer l.mu.Unlock(); return l.held[key] }
