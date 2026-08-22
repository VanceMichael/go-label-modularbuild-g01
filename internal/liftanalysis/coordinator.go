package liftanalysis

import (
	"context"
	"sync"
)

// Coordinator drives the lift-analysis engine. Each plan that enters the
// engine receives its own independently cancellable context, derived from the
// caller's context. Cancelling one plan never cancels a sibling plan that is
// still in flight at a controlled probe point; this was broken previously
// because every plan shared a single context that was torn down on reset.
type Coordinator struct {
	engine Engine

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewCoordinator(engine Engine) *Coordinator {
	return &Coordinator{engine: engine, cancels: make(map[string]context.CancelFunc)}
}

// operationContext opens a per-plan, independently cancellable scope derived
// from the caller's context. The returned cancel frees the scope; callers
// must invoke release when the plan leaves the engine.
func (c *Coordinator) operationContext(caller context.Context, planID string) (context.Context, context.CancelFunc) {
	scoped, cancel := context.WithCancel(caller)
	c.mu.Lock()
	c.cancels[planID] = cancel
	c.mu.Unlock()
	return scoped, cancel
}

// release drops a plan's scope from the coordinator and frees the derived
// context. It touches only the named plan, leaving sibling scopes untouched.
func (c *Coordinator) release(planID string, cancel context.CancelFunc) {
	c.mu.Lock()
	delete(c.cancels, planID)
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Cancel cancels only the context of the named plan. It is safe to call for a
// plan that is not currently in the engine; such a call is a no-op.
func (c *Coordinator) Cancel(planID string) {
	c.mu.Lock()
	cancel, ok := c.cancels[planID]
	c.mu.Unlock()
	if ok {
		cancel()
	}
}
