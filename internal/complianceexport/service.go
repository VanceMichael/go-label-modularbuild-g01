package complianceexport

import (
	"context"
	"fmt"
	"io"
)

type Service struct {
	signer Signer
}

func NewService(signer Signer) *Service { return &Service{signer: signer} }

func (s *Service) Export(ctx context.Context, tenantID string, modules []Module, output io.Writer) error {
	if tenantID == "" || len(modules) == 0 {
		return fmt.Errorf("compliance export: tenant and modules are required")
	}
	encoder := NewEncoder(output)
	if err := encoder.Header(); err != nil {
		return fmt.Errorf("write compliance header: %w", err)
	}
	for _, module := range modules {
		if module.TenantID != tenantID || module.ID == "" || module.Serial == "" || module.SiteCode == "" {
			return fmt.Errorf("compliance export: invalid module %s", module.ID)
		}
		signature, err := s.signer.Sign(ctx, module)
		if err != nil {
			_ = encoder.Flush()
			return fmt.Errorf("sign module %s: %w", module.ID, err)
		}
		if err := encoder.Module(module, signature); err != nil {
			return err
		}
	}
	if err := encoder.Flush(); err != nil {
		return fmt.Errorf("flush compliance export: %w", err)
	}
	return nil
}
