package domain

import "errors"

var (
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("conflict")
	ErrConflictPhone = errors.New("conflict_phone")
	ErrConflictEmail = errors.New("conflict_email")
	ErrInvalidInput  = errors.New("invalid_input")
	ErrUnavailable   = errors.New("dependency_unavailable")
)
