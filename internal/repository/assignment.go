package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

// AssignModuleMove coordinates a lift reservation with the movement update.
func AssignModuleMove(
	ctx context.Context,
	store Store,
	tenant string,
	move domain.ModuleMove,
	window domain.LiftWindow,
	at time.Time,
) (domain.ModuleMove, error) {
	if err := store.ReserveCapacity(ctx, tenant, window.ID, move.WeightKg, window.Version); err != nil {
		return domain.ModuleMove{}, fmt.Errorf("reserve lift capacity: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return domain.ModuleMove{}, err
	}

	move.Status = domain.ModuleMoveBooked
	move.LegID = &window.ID
	move.UpdatedAt = at
	if err := store.UpdateModuleMove(ctx, move, move.Version); err != nil {
		return domain.ModuleMove{}, fmt.Errorf("update module movement: %w", err)
	}
	move.Version++
	return move, nil
}
