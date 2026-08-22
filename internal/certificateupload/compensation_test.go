package certificateupload_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/certificateupload"
)

var errMetadataUnavailable = errors.New("metadata unavailable")

func TestMetadataFailureRemovesStoredCertificateObject(t *testing.T) {
	objects := certificateupload.NewMemoryObjects()
	repo := &certificateupload.MemoryRepository{SaveErr: errMetadataUnavailable}
	service := certificateupload.NewService(objects, repo)
	upload := certificateupload.Upload{
		TenantID: "tenant-site-a", ModuleID: "module-55", Name: "weld-certificate.pdf", Content: []byte("signed-certificate-v3"),
	}

	_, err := service.Upload(context.Background(), upload)
	if !errors.Is(err, errMetadataUnavailable) {
		t.Errorf("Upload() error = %v, want metadata failure", err)
	}
	key := "tenant-site-a/module-55/weld-certificate.pdf"
	if content, ok := objects.Get(key); ok {
		t.Errorf("failed upload left object %q with content %q", key, content)
	}
	if records := repo.Snapshot(); len(records) != 0 {
		t.Errorf("failed upload left metadata: %#v", records)
	}

	validObjects := certificateupload.NewMemoryObjects()
	validRepo := &certificateupload.MemoryRepository{}
	validService := certificateupload.NewService(validObjects, validRepo)
	certificate, err := validService.Upload(context.Background(), upload)
	if err != nil {
		t.Fatalf("valid Upload() error = %v", err)
	}
	content, ok := validObjects.Get(certificate.ObjectKey)
	if !ok || string(content) != string(upload.Content) {
		t.Fatalf("valid object content = %q, ok=%v", content, ok)
	}
	records := validRepo.Snapshot()
	if len(records) != 1 || records[0] != certificate || certificate.Size != len(upload.Content) {
		t.Fatalf("valid metadata = %#v, certificate=%#v", records, certificate)
	}
}
