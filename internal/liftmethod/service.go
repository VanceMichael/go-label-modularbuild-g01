package liftmethod

import (
	"context"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Service struct {
	store  Store
	review Review
	now    func() time.Time
}

func NewService(store Store, review Review) *Service {
	return &Service{store: store, review: review, now: time.Now}
}

func (s *Service) WithClock(now func() time.Time) *Service {
	s.now = now
	return s
}

func (s *Service) Revise(ctx context.Context, tenantID, statementID, title, revision, actorID string, expectedVersion int) error {
	if s.store == nil || s.review == nil || s.now == nil || tenantID == "" || statementID == "" || title == "" || revision == "" || actorID == "" || expectedVersion < 1 {
		return domain.ErrInvalid
	}
	statement, err := s.store.Get(ctx, tenantID, statementID)
	if err != nil {
		return fmt.Errorf("load lift method: %w", err)
	}
	if statement.Version != expectedVersion {
		return domain.ErrConflict
	}
	statement.Title = title
	statement.Revision = revision
	statement.UpdatedBy = actorID
	statement.UpdatedAt = s.now()
	if err := s.review.Check(ctx, statement); err != nil {
		return fmt.Errorf("review lift method: %w", err)
	}
	if err := s.store.Save(ctx, statement, expectedVersion); err != nil {
		return fmt.Errorf("save lift method: %w", err)
	}
	return nil
}
