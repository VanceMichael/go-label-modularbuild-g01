package domain

import "errors"

var (
	ErrNotFound  = errors.New("modularbuild: not found")
	ErrConflict  = errors.New("modularbuild: conflict")
	ErrForbidden = errors.New("modularbuild: forbidden")
	ErrInvalid   = errors.New("modularbuild: invalid input")
	ErrExpired   = errors.New("modularbuild: expired")
	ErrRevoked   = errors.New("modularbuild: revoked")
	ErrCapacity  = errors.New("modularbuild: capacity unavailable")
	ErrState     = errors.New("modularbuild: invalid state transition")
)
