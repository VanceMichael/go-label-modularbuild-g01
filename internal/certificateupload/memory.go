package certificateupload

import (
	"context"
	"sync"
)

type MemoryObjects struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func NewMemoryObjects() *MemoryObjects { return &MemoryObjects{objects: make(map[string][]byte)} }

func (s *MemoryObjects) Put(_ context.Context, key string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = append([]byte(nil), content...)
	return nil
}

func (s *MemoryObjects) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}

func (s *MemoryObjects) Get(key string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, ok := s.objects[key]
	return append([]byte(nil), content...), ok
}

type MemoryRepository struct {
	mu           sync.Mutex
	Certificates []Certificate
	SaveErr      error
}

func (r *MemoryRepository) Save(_ context.Context, certificate Certificate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.SaveErr != nil {
		return r.SaveErr
	}
	r.Certificates = append(r.Certificates, certificate)
	return nil
}

func (r *MemoryRepository) Snapshot() []Certificate {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Certificate(nil), r.Certificates...)
}
