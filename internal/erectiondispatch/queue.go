package erectiondispatch

import (
	"context"
	"fmt"
	"sync"
)

type Queue struct {
	mu    sync.Mutex
	jobs  []Job
	gate  AdmissionGate
	pulse Pulse
}

func NewQueue(gate AdmissionGate, pulse Pulse) *Queue {
	if gate == nil {
		gate = OpenGate{}
	}
	if pulse == nil {
		pulse = NewEdgePulse()
	}
	return &Queue{gate: gate, pulse: pulse}
}

func (q *Queue) Enqueue(job Job) error {
	if job.ID == "" || job.TenantID == "" || job.ModuleID == "" || job.CraneID == "" {
		return ErrInvalidJob
	}

	q.mu.Lock()
	for _, queued := range q.jobs {
		if queued.TenantID == job.TenantID && queued.ID == job.ID {
			q.mu.Unlock()
			return fmt.Errorf("enqueue %s: %w", job.ID, ErrInvalidJob)
		}
	}
	q.jobs = append(q.jobs, job)
	q.mu.Unlock()
	q.pulse.Signal()
	return nil
}

func (q *Queue) Next(ctx context.Context) (Job, error) {
	q.mu.Lock()
	if len(q.jobs) > 0 {
		job := q.popLocked()
		q.mu.Unlock()
		return job, nil
	}
	q.mu.Unlock()

	if err := q.gate.BeforeWait(ctx); err != nil {
		return Job{}, fmt.Errorf("dispatch admission: %w", err)
	}
	if err := q.pulse.Wait(ctx); err != nil {
		return Job{}, fmt.Errorf("wait for dispatch: %w", err)
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.jobs) == 0 {
		return Job{}, ErrNoJob
	}
	return q.popLocked(), nil
}

func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

func (q *Queue) popLocked() Job {
	job := q.jobs[0]
	q.jobs = append([]Job(nil), q.jobs[1:]...)
	return job
}
