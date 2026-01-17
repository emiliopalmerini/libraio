package application

import (
	"errors"
	"fmt"
)

// Sentinel errors for common conditions
var (
	ErrNotFound         = errors.New("not found")
	ErrInvalidID        = errors.New("invalid ID")
	ErrInvalidOperation = errors.New("invalid operation")
	ErrAlreadyArchived  = errors.New("already archived")
	ErrCannotArchive    = errors.New("cannot archive")
)

// ValidationError represents a validation failure with details
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ArchiveError represents an archive-related failure
type ArchiveError struct {
	ID     string
	Reason string
}

func (e *ArchiveError) Error() string {
	return fmt.Sprintf("cannot archive %s: %s", e.ID, e.Reason)
}

func (e *ArchiveError) Is(target error) bool {
	return target == ErrCannotArchive
}

// MoveError represents a move-related failure
type MoveError struct {
	SourceID string
	DestID   string
	Reason   string
}

func (e *MoveError) Error() string {
	return fmt.Sprintf("cannot move %s to %s: %s", e.SourceID, e.DestID, e.Reason)
}
