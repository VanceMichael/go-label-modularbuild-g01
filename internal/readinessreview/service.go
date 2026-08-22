package readinessreview

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	inspector Inspector
	recorder  Recorder
	now       func() time.Time
}

func NewService(inspector Inspector, recorder Recorder, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{inspector: inspector, recorder: recorder, now: now}
}

func (s *Service) ReviewBatch(ctx context.Context, tenantID, batchID string, moduleIDs []string) error {
	if tenantID == "" || batchID == "" || len(moduleIDs) == 0 {
		return fmt.Errorf("readiness review: invalid batch")
	}

	completed := make(chan error, len(moduleIDs))
	for _, moduleID := range moduleIDs {
		moduleID := moduleID
		go func() {
			if err := s.inspector.Inspect(ctx, moduleID); err != nil {
				completed <- fmt.Errorf("inspect module %s: %w", moduleID, err)
				return
			}
			result := Result{
				TenantID: tenantID,
				BatchID:  batchID,
				ModuleID: moduleID,
				Status:   "ready",
				Checked:  s.now(),
			}
			if err := s.recorder.Record(ctx, result); err != nil {
				completed <- fmt.Errorf("record module %s: %w", moduleID, err)
				return
			}
			completed <- nil
		}()
	}

	for range moduleIDs {
		if err := <-completed; err != nil {
			return err
		}
	}
	return nil
}
