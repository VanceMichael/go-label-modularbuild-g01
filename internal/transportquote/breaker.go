package transportquote

import (
	"errors"
	"sync"
)

var ErrCircuitOpen = errors.New("transport quote: circuit open")

type Breaker struct {
	mu                  sync.Mutex
	failures, threshold int
	open                bool
}

func NewBreaker(threshold int) *Breaker {
	if threshold < 1 {
		threshold = 1
	}
	return &Breaker{threshold: threshold}
}
func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.open {
		return ErrCircuitOpen
	}
	return nil
}
func (b *Breaker) Success() { b.mu.Lock(); defer b.mu.Unlock(); b.failures = 0; b.open = false }
func (b *Breaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.threshold {
		b.open = true
	}
}
