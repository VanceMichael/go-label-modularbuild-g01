package repository

import (
	"context"
	"errors"
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
		// Cancellation arrived after the reservation was committed. Release the
		// held capacity so the window keeps zero occupancy and its original
		// version, leaving retry and concurrent callers on a clean slate.
		if rerr := store.ReleaseCapacity(ctx, tenant, window.ID, move.WeightKg, window.Version+1); rerr != nil && !errors.Is(rerr, context.Canceled) {
			return domain.ModuleMove{}, fmt.Errorf("release lift capacity after cancel: %w", rerr)
		}
		return domain.ModuleMove{}, err
	}

	move.Status = domain.ModuleMoveBooked
	move.LegID = &window.ID
	move.UpdatedAt = at
	if err := store.UpdateModuleMove(ctx, move, move.Version); err != nil {
		// The movement update failed after the reservation stuck. Release the
		// held capacity so the window does not leak occupancy to retry calls.
		if rerr := store.ReleaseCapacity(ctx, tenant, window.ID, move.WeightKg, window.Version+1); rerr != nil && !errors.Is(rerr, context.Canceled) {
			return domain.ModuleMove{}, fmt.Errorf("release lift capacity: %w", rerr)
		}
		return domain.ModuleMove{}, fmt.Errorf("update module movement: %w", err)
	}
	move.Version++
	return move, nil
}
