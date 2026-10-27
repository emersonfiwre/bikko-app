package domain

import (
	"errors"
	"fmt"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrConflict  = errors.New("conflict")
	// Keep identity conflict types specific while preserving compatibility with
	// callers that handle all unique-constraint conflicts as ErrConflict.
	ErrConflictPhone = fmt.Errorf("conflict_phone: %w", ErrConflict)
	ErrConflictEmail = fmt.Errorf("conflict_email: %w", ErrConflict)
	ErrInvalidInput  = errors.New("invalid_input")
	ErrUnavailable   = errors.New("dependency_unavailable")
)
