package loadcalculation

import "context"

type Limiter struct{ slots chan struct{} }

func NewLimiter(size int) *Limiter {
	if size < 1 {
		size = 1
	}
	return &Limiter{slots: make(chan struct{}, size)}
}
func (l *Limiter) Acquire(ctx context.Context) error {
	select {
	case l.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (l *Limiter) Release()   { <-l.slots }
func (l *Limiter) InUse() int { return len(l.slots) }
