package workerpool

import (
	"context"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"sort"
	"sync"
)

type Job func(context.Context) error
type Result struct {
	Index int
	Err   error
}

type Pool struct {
	workers int
	jobs    chan Job
	done    chan struct{}
	wg      sync.WaitGroup
	once    sync.Once
}

func New(workers, queue int) *Pool {
	if workers < 1 {
		workers = 1
	}
	if queue < workers {
		queue = workers
	}
	return &Pool{workers: workers, jobs: make(chan Job, queue), done: make(chan struct{})}
}
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-p.done:
					return
				case job := <-p.jobs:
					if job != nil {
						_ = job(ctx)
					}
				}
			}
		}()
	}
}
func (p *Pool) Submit(ctx context.Context, job Job) error {
	if job == nil {
		return domain.ErrInvalid
	}
	select {
	case p.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		return domain.ErrState
	}
}
func (p *Pool) Stop() { p.once.Do(func() { close(p.done); p.wg.Wait() }) }

func Run(ctx context.Context, workers int, jobs []Job) []Result {
	pool := New(workers, len(jobs))
	pool.Start(ctx)
	results := make([]Result, 0, len(jobs))
	var pending sync.WaitGroup
	for index, job := range jobs {
		index, job := index, job
		pending.Add(1)
		err := pool.Submit(ctx, func(jobCtx context.Context) error {
			defer pending.Done()
			err := job(jobCtx)
			results = append(results, Result{Index: index, Err: err})
			return err
		})
		if err != nil {
			pending.Done()
			results = append(results, Result{Index: index, Err: err})
		}
	}
	pending.Wait()
	pool.Stop()
	sort.Slice(results, func(i, j int) bool { return results[i].Index < results[j].Index })
	return results
}
