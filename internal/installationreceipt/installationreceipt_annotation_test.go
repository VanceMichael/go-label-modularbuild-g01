package installationreceipt

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

var errArchiveUnavailable = errors.New("proof archive unavailable")

type annotationArchive struct {
	remaining []error
	calls     int
	lastID    string
	lastProof []byte
}

func (a *annotationArchive) Put(_ context.Context, id string, proof []byte) (string, error) {
	a.calls++
	a.lastID = id
	a.lastProof = append([]byte(nil), proof...)
	if len(a.remaining) > 0 {
		err := a.remaining[0]
		a.remaining = a.remaining[1:]
		if err != nil {
			return "", err
		}
	}
	return "proofs/" + id + ".jpg", nil
}

func TestArchiveFailureDoesNotCompleteInstallation(t *testing.T) {
	now := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	store := NewMemory(Installation{ID: "install-44", TenantID: "tenant-site", Status: "ready", Version: 7})
	archive := &annotationArchive{remaining: []error{errArchiveUnavailable}}
	service := NewService(store, archive, func() time.Time { return now })
	proof := []byte("signed installation photo")

	_, err := service.RecordCompletion(context.Background(), "tenant-site", "install-44", proof)
	if !errors.Is(err, errArchiveUnavailable) {
		t.Fatalf("archive failure error=%v", err)
	}
	afterFailure, err := store.Get(context.Background(), "tenant-site", "install-44")
	if err != nil {
		t.Fatalf("load after failed archive: %v", err)
	}
	if afterFailure.Status != "ready" || afterFailure.Version != 7 || afterFailure.ProofRef != "" {
		t.Fatalf("failed archive changed installation: %+v", afterFailure)
	}

	completed, err := service.RecordCompletion(context.Background(), "tenant-site", "install-44", proof)
	if err != nil {
		t.Fatalf("retry completion: %v", err)
	}
	if completed.Status != "completed" || completed.Version != 8 || completed.ProofRef != "proofs/install-44.jpg" {
		t.Fatalf("retry result=%+v", completed)
	}
	stored, err := store.Get(context.Background(), "tenant-site", "install-44")
	if err != nil || stored != completed {
		t.Fatalf("stored completion=%+v error=%v", stored, err)
	}
	if archive.calls != 2 || archive.lastID != "install-44" || !bytes.Equal(archive.lastProof, proof) {
		t.Fatalf("archive calls=%d id=%q proof=%q", archive.calls, archive.lastID, archive.lastProof)
	}
}
