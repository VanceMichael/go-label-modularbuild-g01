package liftanalysis

import (
	"context"
	"sync"
)

type Coordinator struct {
	engine Engine

	mu        sync.Mutex
	sharedCtx context.Context
	cancel    context.CancelFunc
}

func NewCoordinator(engine Engine) *Coordinator {
	return &Coordinator{engine: engine}
}

func (c *Coordinator) operationContext(caller context.Context) context.Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sharedCtx == nil {
		c.sharedCtx, c.cancel = context.WithCancel(caller)
	}
	return c.sharedCtx
}

func (c *Coordinator) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	c.sharedCtx = nil
	c.cancel = nil
}
