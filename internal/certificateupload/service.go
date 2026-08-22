package certificateupload

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

// ErrMetadataUnavailable is returned when a certificate object has been written
// to the object store but the corresponding repository metadata could not be
// persisted. Callers may match it with errors.Is to detect this partial state.
var ErrMetadataUnavailable = errors.New("modularbuild: certificate metadata unavailable")

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
		// The object was written but its metadata was not persisted. Compensate
		// by deleting the orphaned object so the store and repository stay
		// consistent, and surface the original failure while preserving the
		// errors.Is chain via ErrMetadataUnavailable.
		_ = s.objects.Delete(ctx, key)
		return Certificate{}, fmt.Errorf("%w: save certificate metadata: %w", ErrMetadataUnavailable, err)
	}
	return certificate, nil
}
