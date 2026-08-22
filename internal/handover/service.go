package handover

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	store   *MemoryStore
	checker CertificationChecker
	now     func() time.Time
}

func NewService(store *MemoryStore, checker CertificationChecker, now func() time.Time) *Service {
	return &Service{store: store, checker: checker, now: now}
}

func (s *Service) Complete(ctx context.Context, tenantID, batchID string) (err error) {
	tx, err := s.store.Begin(batchID)
	if err != nil {
		return err
	}
	defer func() {
		if commitErr := tx.Commit(); err == nil && commitErr != nil {
			err = commitErr
		}
	}()

	batch := tx.Batch()
	if batch.TenantID != tenantID {
		return fmt.Errorf("tenant does not own batch")
	}
	if batch.Status != BatchPending {
		return fmt.Errorf("batch is not pending")
	}

	completedAt := s.now()
	for _, moduleID := range batch.ModuleIDs {
		module, ok := tx.Module(moduleID)
		if !ok || module.BatchID != batch.ID || module.Status != ModuleStaged {
			return fmt.Errorf("module %s is not staged for batch", moduleID)
		}
		if checkErr := s.checker.Check(ctx, module); checkErr != nil {
			return fmt.Errorf("certification for module %s: %w", moduleID, checkErr)
		}
		module.Status = ModuleHandedOver
		module.Revision++
		tx.MarkModule(module, AuditEntry{
			BatchID: batch.ID, ModuleID: module.ID, Action: "module_handed_over", At: completedAt,
		})
	}

	batch.Status = BatchCompleted
	batch.CompletedAt = completedAt
	tx.CompleteBatch(batch)
	return nil
}
