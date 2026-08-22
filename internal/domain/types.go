package domain

import "time"

type Role string

const (
	RoleFabricator       Role = "fabricator"
	RoleSitePlanner      Role = "site_planner"
	RoleInstallationCrew Role = "installation_crew"
)

type ModuleMoveStatus string

const (
	ModuleMoveDraft     ModuleMoveStatus = "draft"
	ModuleMoveBooked    ModuleMoveStatus = "booked"
	ModuleMoveScreening ModuleMoveStatus = "screening"
	ModuleMoveCleared   ModuleMoveStatus = "cleared"
	ModuleMoveLoaded    ModuleMoveStatus = "loaded"
	ModuleMoveDeparted  ModuleMoveStatus = "departed"
	ModuleMoveCancelled ModuleMoveStatus = "cancelled"
)

type WindowStatus string

const (
	WindowPlanned  WindowStatus = "planned"
	WindowOpen     WindowStatus = "open"
	WindowBoarding WindowStatus = "boarding"
	WindowDeparted WindowStatus = "departed"
	WindowClosed   WindowStatus = "closed"
)

type QualityStatus string

const (
	QualityPending  QualityStatus = "pending"
	QualityReview   QualityStatus = "review"
	QualityReleased QualityStatus = "released"
	QualityHeld     QualityStatus = "held"
)

type SiteSafetyStatus string

const (
	SiteSafetyPending SiteSafetyStatus = "pending"
	SiteSafetyPassed  SiteSafetyStatus = "passed"
	SiteSafetyFailed  SiteSafetyStatus = "failed"
)

type Tenant struct {
	ID        string
	Name      string
	Active    bool
	CreatedAt time.Time
}
type User struct {
	ID            string
	TenantID      string
	Email         string
	PasswordHash  []byte
	Role          Role
	Active        bool
	CreatedAt     time.Time
	DeactivatedAt *time.Time
}
type Session struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
type ModuleMove struct {
	ID             string
	TenantID       string
	Reference      string
	Origin         string
	Destination    string
	WeightKg       int64
	Pieces         int
	Status         ModuleMoveStatus
	LegID          *string
	IdempotencyKey string
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
type LiftWindow struct {
	ID          string
	TenantID    string
	LiftNumber  string
	Origin      string
	Destination string
	DepartureAt time.Time
	ArrivalAt   time.Time
	CapacityKg  int64
	ReservedKg  int64
	Status      WindowStatus
	Version     int64
	CreatedAt   time.Time
}
type QualityCase struct {
	ID           string
	ModuleMoveID string
	Status       QualityStatus
	DocumentRef  string
	ReviewedBy   *string
	UpdatedAt    time.Time
}
type SiteSafetyCheck struct {
	ID           string
	ModuleMoveID string
	Status       SiteSafetyStatus
	OfficerID    *string
	Notes        string
	CheckedAt    *time.Time
}
type HandlingEvent struct {
	ID           string
	ModuleMoveID string
	Kind         string
	ActorID      string
	OccurredAt   time.Time
	Metadata     map[string]string
}
type AuditEvent struct {
	ID         string
	TenantID   string
	ActorID    string
	ObjectType string
	ObjectID   string
	Action     string
	Result     string
	RequestID  string
	OccurredAt time.Time
}
type OutboxEvent struct {
	ID          string
	TenantID    string
	Topic       string
	AggregateID string
	Payload     []byte
	Attempts    int
	AvailableAt time.Time
	ClaimedAt   *time.Time
	PublishedAt *time.Time
	LastError   string
}
type OperationsSummary struct {
	TenantID      string
	Draft         int
	Booked        int
	InLift        int
	Held          int
	PendingOutbox int
	FailedOutbox  int
}
