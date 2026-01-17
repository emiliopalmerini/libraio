package commands

import (
	"context"
	"fmt"

	"libraio/internal/application"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// DeleteResult contains the result of a delete operation
type DeleteResult struct {
	DeletedID string
	Message   string
}

// DeleteCommand deletes an entity by ID
type DeleteCommand struct {
	repo ports.VaultRepository
	ID   string
}

// NewDeleteCommand creates a new DeleteCommand
func NewDeleteCommand(repo ports.VaultRepository, id string) *DeleteCommand {
	return &DeleteCommand{
		repo: repo,
		ID:   id,
	}
}

// Validate checks if the delete operation is valid
func (c *DeleteCommand) Validate() error {
	if c.ID == "" {
		return &application.ValidationError{
			Field:   "id",
			Message: "ID is required",
		}
	}

	idType := domain.ParseIDType(c.ID)
	if idType == domain.IDTypeUnknown {
		return &application.ValidationError{
			Field:   "id",
			Message: fmt.Sprintf("invalid ID: %s", c.ID),
		}
	}

	return nil
}

// Execute runs the delete command
func (c *DeleteCommand) Execute(ctx context.Context) (*DeleteResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	if err := c.repo.Delete(c.ID); err != nil {
		return nil, fmt.Errorf("failed to delete %s: %w", c.ID, err)
	}

	return &DeleteResult{
		DeletedID: c.ID,
		Message:   fmt.Sprintf("Deleted %s", c.ID),
	}, nil
}
