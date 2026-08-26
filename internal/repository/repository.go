package repository

import (
	"context"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"time"
)

type UserRepository interface {
	GetUserByEmail(context.Context, string) (domain.User, error)
	GetUser(context.Context, string) (domain.User, error)
	CreateUser(context.Context, domain.User) error
	DeactivateUser(context.Context, string, time.Time) error
}
type SessionRepository interface {
	CreateSession(context.Context, domain.Session) error
	GetSession(context.Context, string) (domain.Session, error)
	RevokeSession(context.Context, string, time.Time) error
	RevokeUserSessions(context.Context, string, time.Time) error
}
type ModuleMoveRepository interface {
	CreateModuleMove(context.Context, domain.ModuleMove) error
	GetModuleMove(context.Context, string, string) (domain.ModuleMove, error)
	UpdateModuleMove(context.Context, domain.ModuleMove, int64) error
	ListModuleMoves(context.Context, string, domain.PageRequest) (domain.Page[domain.ModuleMove], error)
	FindByIdempotency(context.Context, string, string) (domain.ModuleMove, error)
}
type LegRepository interface {
	CreateLeg(context.Context, domain.LiftWindow) error
	GetLeg(context.Context, string, string) (domain.LiftWindow, error)
	ReserveCapacity(context.Context, string, string, int64, int64) error
	UpdateWindowStatus(context.Context, string, string, domain.WindowStatus, int64) error
}
type BookingRepository interface {
	BookModuleMove(context.Context, string, string, string, int64, int64, time.Time) (domain.ModuleMove, error)
}
type ComplianceRepository interface {
	GetQuality(context.Context, string) (domain.QualityCase, error)
	PutQuality(context.Context, domain.QualityCase) error
	GetSiteSafety(context.Context, string) (domain.SiteSafetyCheck, error)
	PutSiteSafety(context.Context, domain.SiteSafetyCheck) error
}
type AuditRepository interface {
	AppendAudit(context.Context, domain.AuditEvent) error
	ListAudit(context.Context, string, domain.PageRequest) (domain.Page[domain.AuditEvent], error)
}
type OutboxRepository interface {
	Enqueue(context.Context, domain.OutboxEvent) error
	Claim(context.Context, time.Time, int) ([]domain.OutboxEvent, error)
	MarkPublished(context.Context, string, time.Time) error
	MarkFailed(context.Context, string, time.Time, string) error
	Summary(context.Context, string) (int, int, error)
}
type Store interface {
	UserRepository
	SessionRepository
	ModuleMoveRepository
	LegRepository
	BookingRepository
	ComplianceRepository
	AuditRepository
	OutboxRepository
	Ping(context.Context) error
}
