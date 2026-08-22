package erectiondispatch

import "context"

type EdgePulse struct {
	signal chan struct{}
}

func NewEdgePulse() *EdgePulse {
	return &EdgePulse{signal: make(chan struct{})}
}

func (p *EdgePulse) Signal() {
	select {
	case p.signal <- struct{}{}:
	default:
	}
}

func (p *EdgePulse) Wait(ctx context.Context) error {
	select {
	case <-p.signal:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
