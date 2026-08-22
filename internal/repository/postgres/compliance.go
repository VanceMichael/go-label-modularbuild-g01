package postgres

import (
	"context"
	"errors"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (s *Store) GetQuality(ctx context.Context, id string) (domain.QualityCase, error) {
	var v domain.QualityCase
	var st string
	err := s.db.QueryRow(ctx, `SELECT id,module_move_id,status,document_ref,reviewed_by,updated_at FROM quality_cases WHERE module_move_id=$1`, id).Scan(&v.ID, &v.ModuleMoveID, &st, &v.DocumentRef, &v.ReviewedBy, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, domain.ErrNotFound
	}
	v.Status = domain.QualityStatus(st)
	return v, err
}
func (s *Store) PutQuality(ctx context.Context, v domain.QualityCase) error {
	_, err := s.db.Exec(ctx, `INSERT INTO quality_cases(id,module_move_id,status,document_ref,reviewed_by,updated_at) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(module_move_id) DO UPDATE SET status=EXCLUDED.status,document_ref=EXCLUDED.document_ref,reviewed_by=EXCLUDED.reviewed_by,updated_at=EXCLUDED.updated_at`, v.ID, v.ModuleMoveID, v.Status, v.DocumentRef, v.ReviewedBy, v.UpdatedAt)
	return err
}
func (s *Store) GetSiteSafety(ctx context.Context, id string) (domain.SiteSafetyCheck, error) {
	var v domain.SiteSafetyCheck
	var st string
	err := s.db.QueryRow(ctx, `SELECT id,module_move_id,status,officer_id,notes,checked_at FROM site_safety_checks WHERE module_move_id=$1`, id).Scan(&v.ID, &v.ModuleMoveID, &st, &v.OfficerID, &v.Notes, &v.CheckedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, domain.ErrNotFound
	}
	v.Status = domain.SiteSafetyStatus(st)
	return v, err
}
func (s *Store) PutSiteSafety(ctx context.Context, v domain.SiteSafetyCheck) error {
	_, err := s.db.Exec(ctx, `INSERT INTO site_safety_checks(id,module_move_id,status,officer_id,notes,checked_at) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(module_move_id) DO UPDATE SET status=EXCLUDED.status,officer_id=EXCLUDED.officer_id,notes=EXCLUDED.notes,checked_at=EXCLUDED.checked_at`, v.ID, v.ModuleMoveID, v.Status, v.OfficerID, v.Notes, v.CheckedAt)
	return err
}
