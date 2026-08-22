package manifestimport

import (
	"context"
	"fmt"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Importer struct {
	source Source
	store  Store
}

func NewImporter(source Source, store Store) *Importer {
	return &Importer{source: source, store: store}
}

func (i *Importer) Import(ctx context.Context, tenantID, fileID string) error {
	if i.source == nil || i.store == nil || tenantID == "" || fileID == "" {
		return domain.ErrInvalid
	}
	stream, err := i.source.Open(ctx, tenantID, fileID)
	if err != nil {
		return fmt.Errorf("open manifest source: %w", err)
	}
	err = i.store.Transaction(ctx, func(tx Tx) error {
		for {
			row, readErr := stream.Next(ctx)
			if endOfStream(readErr) {
				return nil
			}
			if readErr != nil {
				return fmt.Errorf("read manifest row: %w", readErr)
			}
			if row.ID == "" || row.TenantID != tenantID || row.ModuleID == "" || row.WeightKg <= 0 {
				return domain.ErrInvalid
			}
			if err := tx.Save(ctx, row); err != nil {
				return fmt.Errorf("save manifest row: %w", err)
			}
		}
	})
	if err != nil {
		return err
	}
	if err := stream.Close(); err != nil {
		return fmt.Errorf("close manifest source: %w", err)
	}
	return nil
}
