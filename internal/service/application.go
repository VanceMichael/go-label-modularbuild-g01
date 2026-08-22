package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/auth"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/repository"
	"log/slog"
	"strings"
	"time"
)

type Application struct {
	store repository.Store
	ttl   time.Duration
	log   *slog.Logger
	clock domain.Clock
}

func NewApplication(store repository.Store, ttl time.Duration, logger *slog.Logger) *Application {
	return &Application{store: store, ttl: ttl, log: logger, clock: domain.RealClock{}}
}
func (a *Application) WithClock(clock domain.Clock) *Application { a.clock = clock; return a }
func (a *Application) Ping(ctx context.Context) error            { return a.store.Ping(ctx) }

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      domain.User
}

func (a *Application) Login(ctx context.Context, email, password string) (LoginResult, error) {
	u, err := a.store.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return LoginResult{}, fmt.Errorf("lookup user: %w", err)
	}
	if !u.Active {
		return LoginResult{}, domain.ErrForbidden
	}
	if err := auth.Compare(u.PasswordHash, password); err != nil {
		return LoginResult{}, domain.ErrForbidden
	}
	raw, err := randomToken()
	if err != nil {
		return LoginResult{}, fmt.Errorf("token: %w", err)
	}
	now := a.clock.Now()
	session := domain.Session{ID: randomID(), UserID: u.ID, TokenHash: auth.HashToken(raw), ExpiresAt: now.Add(a.ttl), CreatedAt: now}
	if err := a.store.CreateSession(ctx, session); err != nil {
		return LoginResult{}, fmt.Errorf("create session: %w", err)
	}
	return LoginResult{Token: raw, ExpiresAt: session.ExpiresAt, User: u}, nil
}
func (a *Application) Logout(ctx context.Context, token string) error {
	if token == "" {
		return domain.ErrInvalid
	}
	if err := a.store.RevokeSession(ctx, token, a.clock.Now()); err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
func (a *Application) Authenticate(ctx context.Context, token string) (domain.User, domain.Session, error) {
	if token == "" {
		return domain.User{}, domain.Session{}, domain.ErrForbidden
	}
	s, err := a.store.GetSession(ctx, token)
	if err != nil {
		return domain.User{}, domain.Session{}, domain.ErrForbidden
	}
	if err := domain.IsSessionActive(s, a.clock.Now()); err != nil {
		return domain.User{}, domain.Session{}, domain.ErrForbidden
	}
	u, err := a.store.GetUser(ctx, s.UserID)
	if err != nil || !u.Active {
		return domain.User{}, domain.Session{}, domain.ErrForbidden
	}
	return u, s, nil
}
func (a *Application) DeactivateUser(ctx context.Context, actor, target string) error {
	u, _, err := a.Authenticate(ctx, actor)
	if err != nil {
		return err
	}
	if u.Role != domain.RoleSitePlanner {
		return domain.ErrForbidden
	}
	if err := a.store.DeactivateUser(ctx, target, a.clock.Now()); err != nil {
		return fmt.Errorf("deactivate: %w", err)
	}
	return nil
}

func (a *Application) CreateModuleMove(ctx context.Context, u domain.User, s domain.ModuleMove, idem string) (domain.ModuleMove, error) {
	if u.Role != domain.RoleFabricator && u.Role != domain.RoleSitePlanner {
		return domain.ModuleMove{}, domain.ErrForbidden
	}
	s.ID = nonEmpty(s.ID, randomID())
	s.TenantID = u.TenantID
	s.Reference = strings.TrimSpace(s.Reference)
	s.Status = domain.ModuleMoveDraft
	s.IdempotencyKey = idem
	s.CreatedAt = a.clock.Now()
	s.UpdatedAt = s.CreatedAt
	s.Version = 1
	if err := domain.ValidateModuleMove(s); err != nil {
		return domain.ModuleMove{}, err
	}
	if idem != "" {
		old, err := a.store.FindByIdempotency(ctx, u.TenantID, idem)
		if err == nil {
			return old, nil
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return domain.ModuleMove{}, err
		}
	}
	if err := a.store.CreateModuleMove(ctx, s); err != nil {
		return domain.ModuleMove{}, fmt.Errorf("create module_move: %w", err)
	}
	return s, nil
}
func (a *Application) GetModuleMove(ctx context.Context, u domain.User, id string) (domain.ModuleMove, error) {
	return a.store.GetModuleMove(ctx, u.TenantID, id)
}
func (a *Application) ListModuleMoves(ctx context.Context, u domain.User, req domain.PageRequest) (domain.Page[domain.ModuleMove], error) {
	if u.Role != domain.RoleFabricator && u.Role != domain.RoleSitePlanner && u.Role != domain.RoleInstallationCrew {
		return domain.Page[domain.ModuleMove]{}, domain.ErrForbidden
	}
	return a.store.ListModuleMoves(ctx, u.TenantID, req)
}
func (a *Application) BookModuleMove(ctx context.Context, u domain.User, id, windowID string) (domain.ModuleMove, error) {
	if u.Role != domain.RoleSitePlanner {
		return domain.ModuleMove{}, domain.ErrForbidden
	}
	s, err := a.store.GetModuleMove(ctx, u.TenantID, id)
	if err != nil {
		return domain.ModuleMove{}, err
	}
	window, err := a.store.GetLeg(ctx, u.TenantID, windowID)
	if err != nil {
		return domain.ModuleMove{}, err
	}
	if !s.CanTransition(domain.ModuleMoveBooked) || window.Status != domain.WindowOpen {
		return domain.ModuleMove{}, domain.ErrState
	}
	if s.Origin != window.Origin || s.Destination != window.Destination {
		return domain.ModuleMove{}, domain.ErrInvalid
	}
	booked, err := a.store.BookModuleMove(ctx, u.TenantID, id, windowID, s.Version, window.Version, a.clock.Now())
	if err != nil {
		return domain.ModuleMove{}, fmt.Errorf("book module_move: %w", err)
	}
	return booked, nil
}
func (a *Application) TransitionModuleMove(ctx context.Context, u domain.User, id string, next domain.ModuleMoveStatus) (domain.ModuleMove, error) {
	if u.Role != domain.RoleSitePlanner && u.Role != domain.RoleInstallationCrew {
		return domain.ModuleMove{}, domain.ErrForbidden
	}
	s, err := a.store.GetModuleMove(ctx, u.TenantID, id)
	if err != nil {
		return domain.ModuleMove{}, err
	}
	if !s.CanTransition(next) {
		return domain.ModuleMove{}, domain.ErrState
	}
	if s.LegID != nil && (next == domain.ModuleMoveLoaded || next == domain.ModuleMoveDeparted) {
		c, ce := a.store.GetQuality(ctx, id)
		sec, se := a.store.GetSiteSafety(ctx, id)
		if ce != nil || se != nil || !domain.QualityAllowsLoading(c) || !domain.SiteSafetyAllowsLoading(sec) {
			return domain.ModuleMove{}, domain.ErrState
		}
	}
	s.Status = next
	s.UpdatedAt = a.clock.Now()
	if err := a.store.UpdateModuleMove(ctx, s, s.Version); err != nil {
		return domain.ModuleMove{}, err
	}
	return s, nil
}

func (a *Application) CreateLeg(ctx context.Context, u domain.User, l domain.LiftWindow) (domain.LiftWindow, error) {
	if u.Role != domain.RoleSitePlanner {
		return domain.LiftWindow{}, domain.ErrForbidden
	}
	l.ID = nonEmpty(l.ID, randomID())
	l.TenantID = u.TenantID
	l.Status = domain.WindowPlanned
	l.Version = 1
	l.CreatedAt = a.clock.Now()
	if err := domain.ValidateLeg(l, a.clock.Now()); err != nil {
		return domain.LiftWindow{}, err
	}
	if err := a.store.CreateLeg(ctx, l); err != nil {
		return domain.LiftWindow{}, err
	}
	return l, nil
}
func (a *Application) OpenLeg(ctx context.Context, u domain.User, id string) (domain.LiftWindow, error) {
	if u.Role != domain.RoleSitePlanner {
		return domain.LiftWindow{}, domain.ErrForbidden
	}
	l, err := a.store.GetLeg(ctx, u.TenantID, id)
	if err != nil {
		return domain.LiftWindow{}, err
	}
	if err := a.store.UpdateWindowStatus(ctx, u.TenantID, id, domain.WindowOpen, l.Version); err != nil {
		return domain.LiftWindow{}, err
	}
	l.Status = domain.WindowOpen
	l.Version++
	return l, nil
}

func (a *Application) CloseLeg(ctx context.Context, u domain.User, id string) (domain.LiftWindow, error) {
	if u.Role != domain.RoleSitePlanner {
		return domain.LiftWindow{}, domain.ErrForbidden
	}
	window, err := a.store.GetLeg(ctx, u.TenantID, id)
	if err != nil {
		return domain.LiftWindow{}, err
	}
	if window.Status != domain.WindowOpen {
		return domain.LiftWindow{}, domain.ErrState
	}
	if window.ReservedKg > 0 {
		return domain.LiftWindow{}, errors.New(domain.ErrState.Error())
	}
	if err := a.store.UpdateWindowStatus(ctx, u.TenantID, id, domain.WindowClosed, window.Version); err != nil {
		return domain.LiftWindow{}, err
	}
	window.Status = domain.WindowClosed
	window.Version++
	return window, nil
}

func (a *Application) PutQuality(ctx context.Context, u domain.User, c domain.QualityCase) (domain.QualityCase, error) {
	if u.Role != domain.RoleSitePlanner {
		return domain.QualityCase{}, domain.ErrForbidden
	}
	if _, err := a.store.GetModuleMove(ctx, u.TenantID, c.ModuleMoveID); err != nil {
		return domain.QualityCase{}, err
	}
	old, err := a.store.GetQuality(ctx, c.ModuleMoveID)
	if err == nil && !old.Status.CanTransition(c.Status) {
		return domain.QualityCase{}, domain.ErrState
	}
	c.ID = nonEmpty(c.ID, randomID())
	c.UpdatedAt = a.clock.Now()
	if err := a.store.PutQuality(ctx, c); err != nil {
		return domain.QualityCase{}, err
	}
	return c, nil
}
func (a *Application) PutSiteSafety(ctx context.Context, u domain.User, s domain.SiteSafetyCheck) (domain.SiteSafetyCheck, error) {
	if u.Role != domain.RoleInstallationCrew && u.Role != domain.RoleSitePlanner {
		return domain.SiteSafetyCheck{}, domain.ErrForbidden
	}
	if _, err := a.store.GetModuleMove(ctx, u.TenantID, s.ModuleMoveID); err != nil {
		return domain.SiteSafetyCheck{}, err
	}
	s.ID = nonEmpty(s.ID, randomID())
	if s.Status == domain.SiteSafetyPassed {
		now := a.clock.Now()
		s.CheckedAt = &now
		s.OfficerID = &u.ID
	}
	if err := a.store.PutSiteSafety(ctx, s); err != nil {
		return domain.SiteSafetyCheck{}, err
	}
	return s, nil
}
func (a *Application) Summary(ctx context.Context, u domain.User) (domain.OperationsSummary, error) {
	if u.Role != domain.RoleSitePlanner {
		return domain.OperationsSummary{}, domain.ErrForbidden
	}
	items, err := a.store.ListModuleMoves(ctx, u.TenantID, domain.PageRequest{Limit: 200})
	if err != nil {
		return domain.OperationsSummary{}, err
	}
	summary := domain.OperationsSummary{TenantID: u.TenantID}
	for _, s := range items.Items {
		switch s.Status {
		case domain.ModuleMoveDraft:
			summary.Draft++
		case domain.ModuleMoveBooked, domain.ModuleMoveScreening, domain.ModuleMoveCleared, domain.ModuleMoveLoaded:
			summary.Booked++
		case domain.ModuleMoveDeparted:
			summary.InLift++
		}
	}
	pending, failed, err := a.store.Summary(ctx, u.TenantID)
	if err != nil {
		return domain.OperationsSummary{}, err
	}
	summary.PendingOutbox = pending
	summary.FailedOutbox = failed
	return summary, nil
}

func randomID() string { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func nonEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
