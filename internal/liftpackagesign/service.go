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

	if err := json.NewEncoder(buffer).Encode(pack); err != nil {
		s.pool.Release(buffer)
		return nil, fmt.Errorf("encode lift package: %w", err)
	}

	// Copy the encoded payload before returning the buffer to the pool. The
	// signer may consume the bytes asynchronously after this method returns,
	// and buffer.Bytes() aliases the pooled buffer's internal storage. Holding
	// that slice while the buffer is released would let a concurrent encode
	// overwrite the payload mid-sign, producing a signature over the wrong data.
	payload := append([]byte(nil), buffer.Bytes()...)
	s.pool.Release(buffer)

	pending, err := s.signer.Start(ctx, pack.PlanID, payload)
	if err != nil {
		return nil, fmt.Errorf("start package signature: %w", err)
	}
	return pending, nil
}
