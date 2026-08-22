package liftexec

import (
	"context"
	"sort"
	"sync"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Memory struct {
	mu           sync.Mutex
	executions   map[string]Execution
	reservations map[string]Reservation
	holds        map[string]SafetyHold
	audits       []AuditEvent
}

func NewMemory() *Memory {
	return &Memory{
		executions:   make(map[string]Execution),
		reservations: make(map[string]Reservation),
		holds:        make(map[string]SafetyHold),
	}
}

func scopedKey(tenantID, id string) string {
	return tenantID + "|" + id
}

func (m *Memory) AddExecution(execution Execution) error {
	if execution.ID == "" || execution.TenantID == "" || execution.ModuleID == "" || execution.ReservationID == "" {
		return domain.ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := scopedKey(execution.TenantID, execution.ID)
	if _, exists := m.executions[key]; exists {
		return domain.ErrConflict
	}
	execution.Status = ExecutionActive
	execution.Version = 1
	m.executions[key] = execution
	return nil
}

func (m *Memory) AddReservation(reservation Reservation) error {
	if reservation.ID == "" || reservation.TenantID == "" || reservation.CraneID == "" {
		return domain.ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := scopedKey(reservation.TenantID, reservation.ID)
	if _, exists := m.reservations[key]; exists {
		return domain.ErrConflict
	}
	reservation.Status = ReservationHeld
	reservation.Version = 1
	m.reservations[key] = reservation
	return nil
}

func (m *Memory) AddHold(hold SafetyHold) error {
	if hold.ID == "" || hold.TenantID == "" || hold.ExecutionID == "" || hold.Reason == "" {
		return domain.ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := scopedKey(hold.TenantID, hold.ID)
	if _, exists := m.holds[key]; exists {
		return domain.ErrConflict
	}
	hold.Status = HoldOpen
	m.holds[key] = hold
	return nil
}

func (m *Memory) ResolveHold(ctx context.Context, tenantID, holdID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := scopedKey(tenantID, holdID)
	hold, exists := m.holds[key]
	if !exists {
		return domain.ErrNotFound
	}
	if hold.Status != HoldOpen {
		return domain.ErrState
	}
	hold.Status = HoldResolved
	m.holds[key] = hold
	return nil
}

func (m *Memory) Transaction(ctx context.Context, fn func(Tx) error) error {
	if fn == nil {
		return domain.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	tx := &memoryTx{
		executions:   cloneMap(m.executions),
		reservations: cloneMap(m.reservations),
		holds:        cloneMap(m.holds),
		audits:       append([]AuditEvent(nil), m.audits...),
	}
	if err := fn(tx); err != nil {
		return err
	}
	m.executions = tx.executions
	m.reservations = tx.reservations
	m.holds = tx.holds
	m.audits = tx.audits
	return nil
}

func cloneMap[K comparable, V any](input map[K]V) map[K]V {
	output := make(map[K]V, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func (m *Memory) GetExecution(ctx context.Context, tenantID, executionID string) (Execution, error) {
	if err := ctx.Err(); err != nil {
		return Execution{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	execution, exists := m.executions[scopedKey(tenantID, executionID)]
	if !exists {
		return Execution{}, domain.ErrNotFound
	}
	return execution, nil
}

func (m *Memory) GetReservation(ctx context.Context, tenantID, reservationID string) (Reservation, error) {
	if err := ctx.Err(); err != nil {
		return Reservation{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	reservation, exists := m.reservations[scopedKey(tenantID, reservationID)]
	if !exists {
		return Reservation{}, domain.ErrNotFound
	}
	return reservation, nil
}

func (m *Memory) AuditCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.audits)
}

type memoryTx struct {
	executions   map[string]Execution
	reservations map[string]Reservation
	holds        map[string]SafetyHold
	audits       []AuditEvent
}

func (tx *memoryTx) GetExecution(ctx context.Context, tenantID, executionID string) (Execution, error) {
	if err := ctx.Err(); err != nil {
		return Execution{}, err
	}
	execution, exists := tx.executions[scopedKey(tenantID, executionID)]
	if !exists {
		return Execution{}, domain.ErrNotFound
	}
	return execution, nil
}

func (tx *memoryTx) GetReservation(ctx context.Context, tenantID, reservationID string) (Reservation, error) {
	if err := ctx.Err(); err != nil {
		return Reservation{}, err
	}
	reservation, exists := tx.reservations[scopedKey(tenantID, reservationID)]
	if !exists {
		return Reservation{}, domain.ErrNotFound
	}
	return reservation, nil
}

func (tx *memoryTx) OpenHolds(ctx context.Context, tenantID, executionID string) ([]SafetyHold, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	holds := make([]SafetyHold, 0)
	for _, hold := range tx.holds {
		if hold.TenantID == tenantID && hold.ExecutionID == executionID && hold.Status == HoldOpen {
			holds = append(holds, hold)
		}
	}
	sort.Slice(holds, func(i, j int) bool { return holds[i].ID < holds[j].ID })
	return holds, nil
}

func (tx *memoryTx) SaveExecution(ctx context.Context, execution Execution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := scopedKey(execution.TenantID, execution.ID)
	if _, exists := tx.executions[key]; !exists {
		return domain.ErrNotFound
	}
	tx.executions[key] = execution
	return nil
}

func (tx *memoryTx) SaveReservation(ctx context.Context, reservation Reservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := scopedKey(reservation.TenantID, reservation.ID)
	if _, exists := tx.reservations[key]; !exists {
		return domain.ErrNotFound
	}
	tx.reservations[key] = reservation
	return nil
}

func (tx *memoryTx) AppendAudit(ctx context.Context, event AuditEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx.audits = append(tx.audits, event)
	return nil
}
