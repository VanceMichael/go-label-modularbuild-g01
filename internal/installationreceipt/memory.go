package installationreceipt

import (
	"context"
	"sync"
	"time"
)

type Memory struct {
	mu            sync.Mutex
	installations map[string]Installation
}

func NewMemory(initial ...Installation) *Memory {
	m := &Memory{installations: make(map[string]Installation, len(initial))}
	for _, installation := range initial {
		m.installations[key(installation.TenantID, installation.ID)] = installation
	}
	return m
}

func (m *Memory) Get(ctx context.Context, tenantID, id string) (Installation, error) {
	if err := ctx.Err(); err != nil {
		return Installation{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	installation, ok := m.installations[key(tenantID, id)]
	if !ok {
		return Installation{}, ErrInstallationMissing
	}
	return installation, nil
}

func (m *Memory) Complete(ctx context.Context, tenantID, id string, version int64, proofRef string, at time.Time) (Installation, error) {
	if err := ctx.Err(); err != nil {
		return Installation{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	installation, ok := m.installations[key(tenantID, id)]
	if !ok {
		return Installation{}, ErrInstallationMissing
	}
	if installation.Status != "ready" || installation.Version != version {
		return Installation{}, ErrInstallationState
	}
	installation.Status = "completed"
	installation.Version++
	installation.ProofRef = proofRef
	installation.UpdatedAt = at
	m.installations[key(tenantID, id)] = installation
	return installation, nil
}

func key(tenantID, id string) string { return tenantID + "/" + id }
