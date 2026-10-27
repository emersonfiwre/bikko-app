package domain

import "errors"

var (
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrInvalidInput = errors.New("invalid_input")
	ErrUnavailable  = errors.New("dependency_unavailable")
)
