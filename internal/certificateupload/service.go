package certificateupload

import (
	"context"
	"fmt"
	"path/filepath"
)

type Service struct {
	objects ObjectStore
	repo    Repository
}

func NewService(objects ObjectStore, repo Repository) *Service {
	return &Service{objects: objects, repo: repo}
}

func (s *Service) Upload(ctx context.Context, upload Upload) (Certificate, error) {
	if upload.TenantID == "" || upload.ModuleID == "" || upload.Name == "" || len(upload.Content) == 0 {
		return Certificate{}, fmt.Errorf("certificate upload: invalid input")
	}
	key := filepath.Join(upload.TenantID, upload.ModuleID, upload.Name)
	if err := s.objects.Put(ctx, key, upload.Content); err != nil {
		return Certificate{}, fmt.Errorf("store certificate object: %w", err)
	}
	certificate := Certificate{
		TenantID: upload.TenantID, ModuleID: upload.ModuleID, Name: upload.Name,
		ObjectKey: key, Size: len(upload.Content),
	}
	if err := s.repo.Save(ctx, certificate); err != nil {
		return Certificate{}, fmt.Errorf("save certificate metadata: %w", err)
	}
	return certificate, nil
}
