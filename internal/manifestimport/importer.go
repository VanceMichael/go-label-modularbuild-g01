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

func (i *Importer) Import(ctx context.Context, tenantID, fileID string) (err error) {
	if i.source == nil || i.store == nil || tenantID == "" || fileID == "" {
		return domain.ErrInvalid
	}
	stream, err := i.source.Open(ctx, tenantID, fileID)
	if err != nil {
		return fmt.Errorf("open manifest source: %w", err)
	}
	// Always close the source once opened so the open count is restored even
	// when the transaction fails partway through (e.g. a read fault on a
	// later row). A transaction error takes precedence over a close error.
	defer func() {
		if cerr := stream.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close manifest source: %w", cerr)
		}
	}()
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
	return err
}
