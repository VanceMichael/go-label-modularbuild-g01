package crane

import (
	"context"
	"sync"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Memory struct {
	mu           sync.RWMutex
	cranes       map[string]Crane
	reservations map[string]Reservation
}

func NewMemory() *Memory {
	return &Memory{cranes: make(map[string]Crane), reservations: make(map[string]Reservation)}
}

func (m *Memory) AddCrane(crane Crane) error {
	if crane.ID == "" || crane.TenantID == "" || crane.CapacityKg <= 0 || crane.ReservedKg < 0 || crane.ReservedKg > crane.CapacityKg {
		return domain.ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.cranes[crane.ID]; exists {
		return domain.ErrConflict
	}
	m.cranes[crane.ID] = crane
	return nil
}

func (m *Memory) GetCrane(_ context.Context, tenantID, craneID string) (Crane, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	crane, exists := m.cranes[craneID]
	if !exists || crane.TenantID != tenantID {
		return Crane{}, domain.ErrNotFound
	}
	return crane, nil
}

func (m *Memory) SaveReservation(_ context.Context, reservation Reservation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.reservations[reservation.ID]; exists {
		return domain.ErrConflict
	}
	crane, exists := m.cranes[reservation.CraneID]
	if !exists || crane.TenantID != reservation.TenantID {
		return domain.ErrNotFound
	}
	crane.ReservedKg += reservation.WeightKg
	m.cranes[crane.ID] = crane
	m.reservations[reservation.ID] = reservation
	return nil
}

func (m *Memory) Reservations(_ context.Context, tenantID, craneID string) ([]Reservation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if crane, exists := m.cranes[craneID]; !exists || crane.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	result := make([]Reservation, 0)
	for _, reservation := range m.reservations {
		if reservation.TenantID == tenantID && reservation.CraneID == craneID {
			result = append(result, reservation)
		}
	}
	return result, nil
}
