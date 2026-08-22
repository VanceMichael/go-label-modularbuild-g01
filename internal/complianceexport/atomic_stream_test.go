package complianceexport_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/complianceexport"
)

var errSigningUnavailable = errors.New("signing unavailable")

type orderedSigner struct {
	failID string
	calls  []string
}

func (s *orderedSigner) Sign(_ context.Context, module complianceexport.Module) (string, error) {
	s.calls = append(s.calls, module.ID)
	if module.ID == s.failID {
		return "", errSigningUnavailable
	}
	return "sig-" + module.ID, nil
}

func TestComplianceExportPublishesOnlyCompleteFile(t *testing.T) {
	modules := []complianceexport.Module{
		{ID: "module-a", TenantID: "tenant-site-a", Serial: "MOD-100", SiteCode: "SITE-7"},
		{ID: "module-b", TenantID: "tenant-site-a", Serial: "MOD-200", SiteCode: "SITE-7"},
	}
	failingSigner := &orderedSigner{failID: "module-b"}
	service := complianceexport.NewService(failingSigner)
	var failedOutput bytes.Buffer

	err := service.Export(context.Background(), "tenant-site-a", modules, &failedOutput)
	if !errors.Is(err, errSigningUnavailable) {
		t.Errorf("Export() error = %v, want signing failure", err)
	}
	if failedOutput.Len() != 0 {
		t.Errorf("failed export published %q, want zero bytes", failedOutput.String())
	}
	if got := strings.Join(failingSigner.calls, ","); got != "module-a,module-b" {
		t.Errorf("signing order = %q", got)
	}

	validSigner := &orderedSigner{}
	validService := complianceexport.NewService(validSigner)
	var validOutput bytes.Buffer
	if err := validService.Export(context.Background(), "tenant-site-a", modules, &validOutput); err != nil {
		t.Fatalf("valid Export() error = %v", err)
	}
	want := "module_id,serial,site_code,signature\n" +
		"module-a,MOD-100,SITE-7,sig-module-a\n" +
		"module-b,MOD-200,SITE-7,sig-module-b\n"
	if validOutput.String() != want {
		t.Fatalf("valid export = %q, want %q", validOutput.String(), want)
	}
	if got := strings.Join(validSigner.calls, ","); got != "module-a,module-b" {
		t.Fatalf("valid signing order = %q", got)
	}
}
