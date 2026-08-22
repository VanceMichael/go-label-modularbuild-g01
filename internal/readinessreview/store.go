package readinessreview

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type Store struct {
	mu      sync.Mutex
	results map[string]Result
}

func NewStore() *Store {
	return &Store{results: make(map[string]Result)}
}

func (s *Store) Record(ctx context.Context, result Result) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if result.TenantID == "" || result.BatchID == "" || result.ModuleID == "" {
		return fmt.Errorf("readiness review: incomplete result")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[result.TenantID+"/"+result.BatchID+"/"+result.ModuleID] = result
	return nil
}

func (s *Store) Results(tenantID, batchID string) []Result {
	s.mu.Lock()
	defer s.mu.Unlock()

	results := make([]Result, 0)
	for _, result := range s.results {
		if result.TenantID == tenantID && result.BatchID == batchID {
			results = append(results, result)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].ModuleID < results[j].ModuleID
	})
	return results
}
