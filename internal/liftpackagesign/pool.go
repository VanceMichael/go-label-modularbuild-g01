package liftpackagesign

import (
	"bytes"
	"context"
)

type Pool struct {
	buffers chan *bytes.Buffer
}

func NewPool(size int) *Pool {
	if size < 1 {
		size = 1
	}
	pool := &Pool{buffers: make(chan *bytes.Buffer, size)}
	for range size {
		pool.buffers <- &bytes.Buffer{}
	}
	return pool
}

func (p *Pool) Acquire(ctx context.Context) (*bytes.Buffer, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case buffer := <-p.buffers:
		return buffer, nil
	}
}

func (p *Pool) Release(buffer *bytes.Buffer) {
	buffer.Reset()
	p.buffers <- buffer
}

func (p *Pool) Available() int {
	return len(p.buffers)
}
