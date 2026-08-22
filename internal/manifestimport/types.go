package manifestimport

import (
	"context"
	"errors"
	"io"
)

var (
	ErrReadFault  = errors.New("manifest source read fault")
	ErrSourceBusy = errors.New("manifest source already open")
)

type Row struct {
	ID       string
	TenantID string
	ModuleID string
	WeightKg int64
}

type Stream interface {
	Next(context.Context) (Row, error)
	Close() error
}

type Source interface {
	Open(context.Context, string, string) (Stream, error)
}

type Tx interface {
	Save(context.Context, Row) error
}

type Store interface {
	Transaction(context.Context, func(Tx) error) error
	Rows(context.Context, string) ([]Row, error)
}

func endOfStream(err error) bool {
	return errors.Is(err, io.EOF)
}
