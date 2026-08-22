package manifestimport

import (
	"context"
	"io"
	"sort"
	"sync"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type sourceFile struct {
	rows       []Row
	failOnceAt int
	open       bool
}

type MemorySource struct {
	mu    sync.Mutex
	files map[string]*sourceFile
}

func NewMemorySource() *MemorySource {
	return &MemorySource{files: make(map[string]*sourceFile)}
}

func sourceKey(tenantID, fileID string) string {
	return tenantID + "|" + fileID
}

func (s *MemorySource) AddFile(tenantID, fileID string, rows []Row, failOnceAt int) error {
	if tenantID == "" || fileID == "" || len(rows) == 0 || failOnceAt < -1 || failOnceAt >= len(rows) {
		return domain.ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sourceKey(tenantID, fileID)
	if _, exists := s.files[key]; exists {
		return domain.ErrConflict
	}
	s.files[key] = &sourceFile{rows: append([]Row(nil), rows...), failOnceAt: failOnceAt}
	return nil
}

func (s *MemorySource) Open(ctx context.Context, tenantID, fileID string) (Stream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	file, exists := s.files[sourceKey(tenantID, fileID)]
	if !exists {
		return nil, domain.ErrNotFound
	}
	if file.open {
		return nil, ErrSourceBusy
	}
	file.open = true
	failAt := file.failOnceAt
	file.failOnceAt = -1
	return &memoryStream{
		source: s,
		file:   file,
		rows:   append([]Row(nil), file.rows...),
		failAt: failAt,
	}, nil
}

func (s *MemorySource) OpenCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, file := range s.files {
		if file.open {
			count++
		}
	}
	return count
}

type memoryStream struct {
	source *MemorySource
	file   *sourceFile
	rows   []Row
	index  int
	failAt int
	closed bool
}

func (s *memoryStream) Next(ctx context.Context) (Row, error) {
	if err := ctx.Err(); err != nil {
		return Row{}, err
	}
	if s.closed {
		return Row{}, domain.ErrState
	}
	if s.index == s.failAt {
		s.failAt = -1
		return Row{}, ErrReadFault
	}
	if s.index >= len(s.rows) {
		return Row{}, io.EOF
	}
	row := s.rows[s.index]
	s.index++
	return row, nil
}

func (s *memoryStream) Close() error {
	s.source.mu.Lock()
	defer s.source.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	s.file.open = false
	return nil
}

type MemoryStore struct {
	mu   sync.Mutex
	rows map[string]Row
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: make(map[string]Row)}
}

func rowKey(tenantID, rowID string) string {
	return tenantID + "|" + rowID
}

func (s *MemoryStore) Transaction(ctx context.Context, fn func(Tx) error) error {
	if fn == nil {
		return domain.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make(map[string]Row, len(s.rows))
	for key, row := range s.rows {
		rows[key] = row
	}
	tx := &memoryTx{rows: rows}
	if err := fn(tx); err != nil {
		return err
	}
	s.rows = tx.rows
	return nil
}

func (s *MemoryStore) Rows(ctx context.Context, tenantID string) ([]Row, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]Row, 0)
	for _, row := range s.rows {
		if row.TenantID == tenantID {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows, nil
}

type memoryTx struct {
	rows map[string]Row
}

func (tx *memoryTx) Save(ctx context.Context, row Row) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := rowKey(row.TenantID, row.ID)
	if _, exists := tx.rows[key]; exists {
		return domain.ErrConflict
	}
	tx.rows[key] = row
	return nil
}
