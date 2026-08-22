package leasereaper

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	store   Store
	barrier Barrier
	now     func() time.Time
}

func NewService(store Store, barrier Barrier, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, barrier: barrier, now: now}
}

func (s *Service) Reap(ctx context.Context) error {
	leases, err := s.store.Expired(ctx, s.now())
	if err != nil {
		return fmt.Errorf("scan expired leases: %w", err)
	}
	if s.barrier != nil {
		if err := s.barrier.AfterScan(ctx); err != nil {
			return fmt.Errorf("lease cleanup barrier: %w", err)
		}
	}
	for _, lease := range leases {
		if err := s.store.Delete(ctx, lease.TenantID, lease.Resource); err != nil {
			return fmt.Errorf("delete expired lease: %w", err)
		}
	}
	return nil
}
