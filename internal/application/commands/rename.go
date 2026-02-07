package commands

import (
	"context"
	"fmt"
	"strings"

	"libraio/internal/application"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// RenameResult contains the result of a rename operation
type RenameResult struct {
	OriginalID string
	NewName    string
	Message    string
}

// RenameCommand renames an item, category, or area
type RenameCommand struct {
	repo           ports.VaultRepository
	ID             string
	NewDescription string
}

// NewRenameCommand creates a new RenameCommand
func NewRenameCommand(repo ports.VaultRepository, id, newDescription string) *RenameCommand {
	return &RenameCommand{
		repo:           repo,
		ID:             id,
		NewDescription: newDescription,
	}
}

// Validate checks if the rename operation is valid
func (c *RenameCommand) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return &application.ValidationError{
			Field:   "id",
			Message: "ID is required",
		}
	}

	if strings.TrimSpace(c.NewDescription) == "" {
		return &application.ValidationError{
			Field:   "description",
			Message: "description is required",
		}
	}

	idType := domain.ParseIDType(c.ID)
	switch idType {
	case domain.IDTypeItem, domain.IDTypeCategory, domain.IDTypeArea:
		return nil
	case domain.IDTypeScope:
		return &application.ValidationError{
			Field:   "id",
			Message: "cannot rename scopes",
		}
	default:
		return &application.ValidationError{
			Field:   "id",
			Message: fmt.Sprintf("invalid ID: %s", c.ID),
		}
	}
}

// Execute runs the rename command
func (c *RenameCommand) Execute(ctx context.Context) (*RenameResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	newDescription := strings.TrimSpace(c.NewDescription)
	idType := domain.ParseIDType(c.ID)

	var err error
	switch idType {
	case domain.IDTypeItem:
		_, err = c.repo.RenameItem(c.ID, newDescription)
	case domain.IDTypeCategory:
		_, err = c.repo.RenameCategory(c.ID, newDescription)
	case domain.IDTypeArea:
		_, err = c.repo.RenameArea(c.ID, newDescription)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to rename: %w", err)
	}

	return &RenameResult{
		OriginalID: c.ID,
		NewName:    newDescription,
		Message:    fmt.Sprintf("Renamed %s to %s", c.ID, newDescription),
	}, nil
}

// RenameEligibility contains the result of checking if a node can be renamed
type RenameEligibility struct {
	CanRename bool
	Reason    string
}

// CheckRenameEligibility determines if a node type can be renamed
func CheckRenameEligibility(nodeType domain.IDType) RenameEligibility {
	switch nodeType {
	case domain.IDTypeItem, domain.IDTypeCategory, domain.IDTypeArea:
		return RenameEligibility{CanRename: true}
	default:
		return RenameEligibility{
			CanRename: false,
			Reason:    fmt.Sprintf("cannot rename %s", nodeType),
		}
	}
}
