package commands

import (
	"context"
	"fmt"

	"libraio/internal/application"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// MoveItemResult contains the result of moving an item
type MoveItemResult struct {
	OriginalID string
	MovedItem  *domain.Item
	Message    string
}

// MoveItemCommand moves an item to a different category
type MoveItemCommand struct {
	repo             ports.VaultRepository
	SourceItemID     string
	DestinationCatID string
}

// NewMoveItemCommand creates a new MoveItemCommand
func NewMoveItemCommand(repo ports.VaultRepository, sourceItemID, destCategoryID string) *MoveItemCommand {
	return &MoveItemCommand{
		repo:             repo,
		SourceItemID:     sourceItemID,
		DestinationCatID: destCategoryID,
	}
}

// Validate checks if the move operation is valid
func (c *MoveItemCommand) Validate() error {
	if c.SourceItemID == "" {
		return &application.ValidationError{
			Field:   "sourceItemID",
			Message: "source item ID is required",
		}
	}

	if c.DestinationCatID == "" {
		return &application.ValidationError{
			Field:   "destinationCategoryID",
			Message: "destination category ID is required",
		}
	}

	srcType := domain.ParseIDType(c.SourceItemID)
	if srcType != domain.IDTypeItem {
		return &application.MoveError{
			SourceID: c.SourceItemID,
			DestID:   c.DestinationCatID,
			Reason:   fmt.Sprintf("source must be an item, got: %s", srcType),
		}
	}

	destType := domain.ParseIDType(c.DestinationCatID)
	if destType != domain.IDTypeCategory {
		return &application.MoveError{
			SourceID: c.SourceItemID,
			DestID:   c.DestinationCatID,
			Reason:   "items can only be moved to categories",
		}
	}

	return nil
}

// Execute runs the move item command
func (c *MoveItemCommand) Execute(ctx context.Context) (*MoveItemResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	item, err := c.repo.MoveItem(c.SourceItemID, c.DestinationCatID)
	if err != nil {
		return nil, fmt.Errorf("failed to move item: %w", err)
	}

	return &MoveItemResult{
		OriginalID: c.SourceItemID,
		MovedItem:  item,
		Message:    fmt.Sprintf("Moved to %s %s", item.ID, item.Name),
	}, nil
}

// MoveCategoryResult contains the result of moving a category
type MoveCategoryResult struct {
	OriginalID    string
	MovedCategory *domain.Category
	Message       string
}

// MoveCategoryCommand moves a category to a different area
type MoveCategoryCommand struct {
	repo            ports.VaultRepository
	SourceCatID     string
	DestinationArea string
}

// NewMoveCategoryCommand creates a new MoveCategoryCommand
func NewMoveCategoryCommand(repo ports.VaultRepository, sourceCatID, destAreaID string) *MoveCategoryCommand {
	return &MoveCategoryCommand{
		repo:            repo,
		SourceCatID:     sourceCatID,
		DestinationArea: destAreaID,
	}
}

// Validate checks if the move operation is valid
func (c *MoveCategoryCommand) Validate() error {
	if c.SourceCatID == "" {
		return &application.ValidationError{
			Field:   "sourceCategoryID",
			Message: "source category ID is required",
		}
	}

	if c.DestinationArea == "" {
		return &application.ValidationError{
			Field:   "destinationAreaID",
			Message: "destination area ID is required",
		}
	}

	srcType := domain.ParseIDType(c.SourceCatID)
	if srcType != domain.IDTypeCategory {
		return &application.MoveError{
			SourceID: c.SourceCatID,
			DestID:   c.DestinationArea,
			Reason:   fmt.Sprintf("source must be a category, got: %s", srcType),
		}
	}

	destType := domain.ParseIDType(c.DestinationArea)
	if destType != domain.IDTypeArea {
		return &application.MoveError{
			SourceID: c.SourceCatID,
			DestID:   c.DestinationArea,
			Reason:   "categories can only be moved to areas",
		}
	}

	return nil
}

// Execute runs the move category command
func (c *MoveCategoryCommand) Execute(ctx context.Context) (*MoveCategoryResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	cat, err := c.repo.MoveCategory(c.SourceCatID, c.DestinationArea)
	if err != nil {
		return nil, fmt.Errorf("failed to move category: %w", err)
	}

	return &MoveCategoryResult{
		OriginalID:    c.SourceCatID,
		MovedCategory: cat,
		Message:       fmt.Sprintf("Moved to %s %s", cat.ID, cat.Name),
	}, nil
}

// ValidateMoveDestination checks if a move operation is valid without executing it
func ValidateMoveDestination(sourceID string, sourceType domain.IDType, destID string) error {
	destType := domain.ParseIDType(destID)

	switch sourceType {
	case domain.IDTypeItem:
		if destType != domain.IDTypeCategory {
			return &application.MoveError{
				SourceID: sourceID,
				DestID:   destID,
				Reason:   "items can only be moved to categories",
			}
		}
	case domain.IDTypeCategory:
		if destType != domain.IDTypeArea {
			return &application.MoveError{
				SourceID: sourceID,
				DestID:   destID,
				Reason:   "categories can only be moved to areas",
			}
		}
	default:
		return &application.MoveError{
			SourceID: sourceID,
			DestID:   destID,
			Reason:   fmt.Sprintf("cannot move %s", sourceType),
		}
	}

	return nil
}
