package postgres

import (
	"context"
	"errors"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *Store) CreateModuleMove(ctx context.Context, v domain.ModuleMove) error {
	_, err := s.db.Exec(ctx, `INSERT INTO module_moves(id,tenant_id,reference,origin,destination,weight_kg,pieces,status,window_id,idempotency_key,version,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, v.ID, v.TenantID, v.Reference, v.Origin, v.Destination, v.WeightKg, v.Pieces, v.Status, v.LegID, v.IdempotencyKey, v.Version, v.CreatedAt, v.UpdatedAt)
	if isUnique(err) {
		return domain.ErrConflict
	}
	return err
}
func (s *Store) GetModuleMove(ctx context.Context, tenant, id string) (domain.ModuleMove, error) {
	return s.module_move(ctx, `SELECT id,tenant_id,reference,origin,destination,weight_kg,pieces,status,window_id,idempotency_key,version,created_at,updated_at FROM module_moves WHERE tenant_id=$1 AND id=$2`, tenant, id)
}
func (s *Store) module_move(ctx context.Context, q string, args ...any) (domain.ModuleMove, error) {
	var v domain.ModuleMove
	var status string
	err := s.db.QueryRow(ctx, q, args...).Scan(&v.ID, &v.TenantID, &v.Reference, &v.Origin, &v.Destination, &v.WeightKg, &v.Pieces, &status, &v.LegID, &v.IdempotencyKey, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, domain.ErrNotFound
	}
	v.Status = domain.ModuleMoveStatus(status)
	return v, err
}
func (s *Store) UpdateModuleMove(ctx context.Context, v domain.ModuleMove, version int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE module_moves SET reference=$3,origin=$4,destination=$5,weight_kg=$6,pieces=$7,status=$8,window_id=$9,updated_at=$10,version=version+1 WHERE tenant_id=$1 AND id=$2 AND version=$11`, v.TenantID, v.ID, v.Reference, v.Origin, v.Destination, v.WeightKg, v.Pieces, v.Status, v.LegID, v.UpdatedAt, version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (s *Store) ListModuleMoves(ctx context.Context, tenant string, req domain.PageRequest) (domain.Page[domain.ModuleMove], error) {
	req = req.Normalized()
	rows, err := s.db.Query(ctx, `SELECT id,tenant_id,reference,origin,destination,weight_kg,pieces,status,window_id,idempotency_key,version,created_at,updated_at FROM module_moves WHERE tenant_id=$1 ORDER BY created_at,id LIMIT $2`, tenant, req.Limit)
	if err != nil {
		return domain.Page[domain.ModuleMove]{}, err
	}
	defer rows.Close()
	out := make([]domain.ModuleMove, 0)
	for rows.Next() {
		var v domain.ModuleMove
		var st string
		if err := rows.Scan(&v.ID, &v.TenantID, &v.Reference, &v.Origin, &v.Destination, &v.WeightKg, &v.Pieces, &st, &v.LegID, &v.IdempotencyKey, &v.Version, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return domain.Page[domain.ModuleMove]{}, err
		}
		v.Status = domain.ModuleMoveStatus(st)
		out = append(out, v)
	}
	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM module_moves WHERE tenant_id=$1`, tenant).Scan(&total); err != nil {
		return domain.Page[domain.ModuleMove]{}, err
	}
	return domain.Page[domain.ModuleMove]{Items: out, Total: total}, rows.Err()
}
func (s *Store) FindByIdempotency(ctx context.Context, tenant, key string) (domain.ModuleMove, error) {
	return s.module_move(ctx, `SELECT id,tenant_id,reference,origin,destination,weight_kg,pieces,status,window_id,idempotency_key,version,created_at,updated_at FROM module_moves WHERE tenant_id=$1 AND idempotency_key=$2`, tenant, key)
}
func (s *Store) CreateLeg(ctx context.Context, v domain.LiftWindow) error {
	_, err := s.db.Exec(ctx, `INSERT INTO lift_windows(id,tenant_id,flight_number,origin,destination,departure_at,arrival_at,capacity_kg,reserved_kg,status,version,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, v.ID, v.TenantID, v.LiftNumber, v.Origin, v.Destination, v.DepartureAt, v.ArrivalAt, v.CapacityKg, v.ReservedKg, v.Status, v.Version, v.CreatedAt)
	return err
}
func (s *Store) GetLeg(ctx context.Context, tenant, id string) (domain.LiftWindow, error) {
	var v domain.LiftWindow
	var st string
	err := s.db.QueryRow(ctx, `SELECT id,tenant_id,flight_number,origin,destination,departure_at,arrival_at,capacity_kg,reserved_kg,status,version,created_at FROM lift_windows WHERE tenant_id=$1 AND id=$2`, tenant, id).Scan(&v.ID, &v.TenantID, &v.LiftNumber, &v.Origin, &v.Destination, &v.DepartureAt, &v.ArrivalAt, &v.CapacityKg, &v.ReservedKg, &st, &v.Version, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, domain.ErrNotFound
	}
	v.Status = domain.WindowStatus(st)
	return v, err
}
func (s *Store) ReserveCapacity(ctx context.Context, tenant, id string, weight, version int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE lift_windows SET reserved_kg=reserved_kg+$4,version=version+1 WHERE tenant_id=$1 AND id=$2 AND version=$3 AND reserved_kg+$4<=capacity_kg`, tenant, id, version, weight)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (s *Store) ReleaseCapacity(ctx context.Context, tenant, id string, weight, version int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE lift_windows SET reserved_kg=reserved_kg-$4,version=version-1 WHERE tenant_id=$1 AND id=$2 AND version=$3 AND reserved_kg>=$4`, tenant, id, version, weight)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (s *Store) UpdateWindowStatus(ctx context.Context, tenant, id string, status domain.WindowStatus, version int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE lift_windows SET status=$4,version=version+1 WHERE tenant_id=$1 AND id=$2 AND version=$3`, tenant, id, version, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}

var _ = time.UTC
