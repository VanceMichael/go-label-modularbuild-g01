package permit

import (
	"context"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Service struct {
	store Store
	cache Cache
	now   func() time.Time
}

func NewService(store Store, cache Cache) *Service {
	return &Service{store: store, cache: cache, now: time.Now}
}

func (s *Service) WithClock(now func() time.Time) *Service {
	s.now = now
	return s
}

func (s *Service) Approve(ctx context.Context, tenantID, permitID, actorID string) error {
	if s.store == nil || s.cache == nil || s.now == nil || tenantID == "" || permitID == "" || actorID == "" {
		return domain.ErrInvalid
	}
	var approved Permit
	if err := s.store.Transaction(ctx, func(tx Tx) error {
		permit, err := tx.GetPermit(ctx, tenantID, permitID)
		if err != nil {
			return fmt.Errorf("load permit: %w", err)
		}
		if permit.Status != StatusPending {
			return domain.ErrState
		}
		approvedAt := s.now()
		permit.Status = StatusApproved
		permit.ApprovedBy = actorID
		permit.ApprovedAt = &approvedAt
		permit.Version++
		if err := tx.SavePermit(ctx, permit); err != nil {
			return fmt.Errorf("save permit: %w", err)
		}
		if err := tx.AppendAudit(ctx, AuditEvent{
			PermitID: permit.ID,
			TenantID: permit.TenantID,
			ActorID:  actorID,
			Action:   "permit.approved",
			At:       approvedAt,
		}); err != nil {
			return fmt.Errorf("append permit audit: %w", err)
		}
		approved = permit
		return nil
	}); err != nil {
		return err
	}
	// Publish readiness only after the transaction commits so an audit
	// failure rolls the permit back without leaving a stale cached record.
	if err := s.cache.Put(ctx, approved); err != nil {
		return fmt.Errorf("publish permit readiness: %w", err)
	}
	return nil
}

func (s *Service) CanDispatch(ctx context.Context, tenantID, permitID string) (bool, error) {
	if s.store == nil || s.cache == nil || tenantID == "" || permitID == "" {
		return false, domain.ErrInvalid
	}
	permit, err := s.cache.Get(ctx, tenantID, permitID)
	if err == nil {
		return permit.Status == StatusApproved, nil
	}
	if err != domain.ErrNotFound {
		return false, fmt.Errorf("read permit readiness: %w", err)
	}
	permit, err = s.store.GetPermit(ctx, tenantID, permitID)
	if err != nil {
		return false, fmt.Errorf("load permit: %w", err)
	}
	return permit.Status == StatusApproved, nil
}
