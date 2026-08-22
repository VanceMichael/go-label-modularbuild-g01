package certificateupload

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type recorderSave struct {
	saved []Certificate
	err   error
}

func (r *recorderSave) Save(_ context.Context, c Certificate) error {
	if r.err != nil {
		return r.err
	}
	r.saved = append(r.saved, c)
	return nil
}

func TestUploadMetadataUnavailableCompensatesAndPreservesErrorChain(t *testing.T) {
	objects := NewMemoryObjects()
	repo := &recorderSave{err: errors.New("db: metadata unavailable")}
	svc := NewService(objects, repo)

	upload := Upload{
		TenantID: "tenant-site-a",
		ModuleID: "module-55",
		Name:     "weld-certificate.pdf",
		Content:  []byte("signed-certificate-v3"),
	}
	_, err := svc.Upload(context.Background(), upload)
	if err == nil {
		t.Fatal("expected upload to fail when metadata save fails")
	}
	if !errors.Is(err, ErrMetadataUnavailable) {
		t.Fatalf("expected errors.Is(err, ErrMetadataUnavailable), got %v", err)
	}
	if !errors.Is(err, repo.err) {
		t.Fatalf("expected errors.Is(err, underlying save error), got %v", err)
	}
	if !strings.Contains(err.Error(), "save certificate metadata") {
		t.Fatalf("expected wrapped context in message, got %q", err.Error())
	}

	// The orphaned object must be deleted and the repository must stay empty.
	if got, ok := objects.Get("tenant-site-a/module-55/weld-certificate.pdf"); ok || got != nil {
		t.Fatalf("expected object to be deleted, found %v", got)
	}
	if objects.DeleteCalls != 1 {
		t.Fatalf("expected exactly one compensating Delete call, got %d", objects.DeleteCalls)
	}
	if len(repo.saved) != 0 {
		t.Fatalf("expected repository to remain empty, got %d records", len(repo.saved))
	}
}
