package liftpackagesign

import (
	"context"
	"encoding/json"
	"fmt"
)

type Service struct {
	pool   BufferPool
	signer Signer
}

func NewService(pool BufferPool, signer Signer) *Service {
	return &Service{pool: pool, signer: signer}
}

func (s *Service) RenderAndSign(ctx context.Context, pack Package) (SignedArtifact, error) {
	if pack.TenantID == "" || pack.PlanID == "" || len(pack.Modules) == 0 {
		return SignedArtifact{}, ErrInvalidPackage
	}

	pending, err := s.prepare(ctx, pack)
	if err != nil {
		return SignedArtifact{}, err
	}
	signed, err := pending.Wait(ctx)
	if err != nil {
		return SignedArtifact{}, fmt.Errorf("wait for package signature: %w", err)
	}
	return signed, nil
}

func (s *Service) prepare(ctx context.Context, pack Package) (Pending, error) {
	buffer, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire package encoder: %w", err)
	}
	defer s.pool.Release(buffer)

	if err := json.NewEncoder(buffer).Encode(pack); err != nil {
		return nil, fmt.Errorf("encode lift package: %w", err)
	}
	pending, err := s.signer.Start(ctx, pack.PlanID, buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("start package signature: %w", err)
	}
	return pending, nil
}
