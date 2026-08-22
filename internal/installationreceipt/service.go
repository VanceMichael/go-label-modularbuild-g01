package installationreceipt

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	store   Store
	archive Archive
	now     func() time.Time
}

func NewService(store Store, archive Archive, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, archive: archive, now: now}
}

func (s *Service) RecordCompletion(ctx context.Context, tenantID, installationID string, proof []byte) (Installation, error) {
	installation, err := s.store.Get(ctx, tenantID, installationID)
	if err != nil {
		return Installation{}, fmt.Errorf("load installation: %w", err)
	}
	completed, err := s.store.Complete(ctx, tenantID, installationID, installation.Version, "", s.now())
	if err != nil {
		return Installation{}, fmt.Errorf("complete installation: %w", err)
	}
	proofRef, err := s.archive.Put(ctx, installationID, proof)
	if err != nil {
		return Installation{}, fmt.Errorf("archive installation proof: %w", err)
	}
	completed.ProofRef = proofRef
	return completed, nil
}
