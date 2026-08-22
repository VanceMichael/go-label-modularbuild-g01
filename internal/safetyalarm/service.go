package safetyalarm

import (
	"context"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Service struct {
	registry *Registry
}

func NewService(registry *Registry) *Service {
	return &Service{registry: registry}
}

func (s *Service) Publish(ctx context.Context, alarm Alarm) error {
	if s.registry == nil || alarm.ID == "" || alarm.TenantID == "" || alarm.CraneID == "" || alarm.Kind == "" || alarm.ObservedAt.IsZero() {
		return domain.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.registry.Dispatch(ctx, alarm)
}
