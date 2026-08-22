package dispatch

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Memory struct {
	mu   sync.RWMutex
	jobs map[string]Job
}

func NewMemory() *Memory {
	return &Memory{jobs: make(map[string]Job)}
}

func (m *Memory) Add(job Job) error {
	if job.ID == "" || job.TenantID == "" || job.ModuleMoveID == "" {
		return domain.ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.jobs[job.ID]; exists {
		return domain.ErrConflict
	}
	job.Status = StatusPending
	job.Attempts = 0
	job.LeaseUntil = nil
	m.jobs[job.ID] = job
	return nil
}

func (m *Memory) ClaimPending(_ context.Context, tenantID string, now time.Time, lease time.Duration) (Job, error) {
	if tenantID == "" || now.IsZero() || lease <= 0 {
		return Job{}, domain.ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0)
	for id, job := range m.jobs {
		if job.TenantID == tenantID && job.Status == StatusPending {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return Job{}, domain.ErrNotFound
	}
	sort.Strings(ids)
	job := m.jobs[ids[0]]
	expires := now.Add(lease)
	job.Status = StatusRunning
	job.Attempts++
	job.LeaseUntil = &expires
	m.jobs[job.ID] = job
	return job, nil
}

func (m *Memory) Complete(_ context.Context, id string, attempt int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, exists := m.jobs[id]
	if !exists {
		return domain.ErrNotFound
	}
	if job.Status != StatusRunning || job.Attempts != attempt {
		return domain.ErrConflict
	}
	job.Status = StatusDone
	job.LeaseUntil = nil
	m.jobs[id] = job
	return nil
}

func (m *Memory) Requeue(_ context.Context, id string, attempt int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, exists := m.jobs[id]
	if !exists {
		return domain.ErrNotFound
	}
	if job.Status != StatusRunning || job.Attempts != attempt {
		return domain.ErrConflict
	}
	job.Status = StatusPending
	job.LeaseUntil = nil
	m.jobs[id] = job
	return nil
}

func (m *Memory) Get(_ context.Context, tenantID, id string) (Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, exists := m.jobs[id]
	if !exists || job.TenantID != tenantID {
		return Job{}, domain.ErrNotFound
	}
	return job, nil
}
