package postgres

import (
	"context"
	"errors"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *Store) BookModuleMove(ctx context.Context, tenant, module_moveID, windowID string, module_moveVersion, windowVersion int64, at time.Time) (domain.ModuleMove, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.ModuleMove{}, err
	}
	defer tx.Rollback(ctx)
	var module_move domain.ModuleMove
	var module_moveStatus string
	err = tx.QueryRow(ctx, `SELECT id,tenant_id,reference,origin,destination,weight_kg,pieces,status,window_id,idempotency_key,version,created_at,updated_at FROM module_moves WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenant, module_moveID).Scan(&module_move.ID, &module_move.TenantID, &module_move.Reference, &module_move.Origin, &module_move.Destination, &module_move.WeightKg, &module_move.Pieces, &module_moveStatus, &module_move.LegID, &module_move.IdempotencyKey, &module_move.Version, &module_move.CreatedAt, &module_move.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ModuleMove{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ModuleMove{}, err
	}
	module_move.Status = domain.ModuleMoveStatus(module_moveStatus)
	var window domain.LiftWindow
	var windowStatus string
	err = tx.QueryRow(ctx, `SELECT id,tenant_id,flight_number,origin,destination,departure_at,arrival_at,capacity_kg,reserved_kg,status,version,created_at FROM lift_windows WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenant, windowID).Scan(&window.ID, &window.TenantID, &window.LiftNumber, &window.Origin, &window.Destination, &window.DepartureAt, &window.ArrivalAt, &window.CapacityKg, &window.ReservedKg, &windowStatus, &window.Version, &window.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ModuleMove{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ModuleMove{}, err
	}
	window.Status = domain.WindowStatus(windowStatus)
	if module_move.Version != module_moveVersion || window.Version != windowVersion {
		return domain.ModuleMove{}, domain.ErrConflict
	}
	if !module_move.CanTransition(domain.ModuleMoveBooked) || window.Status != domain.WindowOpen {
		return domain.ModuleMove{}, domain.ErrState
	}
	if module_move.Origin != window.Origin || module_move.Destination != window.Destination {
		return domain.ModuleMove{}, domain.ErrInvalid
	}
	if err := domain.ValidateCapacity(window, module_move.WeightKg); err != nil {
		return domain.ModuleMove{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE lift_windows SET reserved_kg=reserved_kg+$3,version=version+1 WHERE tenant_id=$1 AND id=$2`, tenant, windowID, module_move.WeightKg); err != nil {
		return domain.ModuleMove{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE module_moves SET status=$3,window_id=$4,updated_at=$5,version=version+1 WHERE tenant_id=$1 AND id=$2`, tenant, module_moveID, domain.ModuleMoveBooked, windowID, at); err != nil {
		return domain.ModuleMove{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.ModuleMove{}, err
	}
	module_move.Status = domain.ModuleMoveBooked
	module_move.LegID = &windowID
	module_move.UpdatedAt = at
	module_move.Version++
	return module_move, nil
}
