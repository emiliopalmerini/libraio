package commands

import (
	"context"
	"fmt"

	"libraio/internal/application"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// UnarchiveItemResult contains the result of unarchiving an item
type UnarchiveItemResult struct {
	ArchiveItemID string
	RestoredItems []string
	Message       string
}

// UnarchiveItemCommand restores archived items from an archive folder back to their category
type UnarchiveItemCommand struct {
	repo          ports.VaultRepository
	ArchiveItemID string // The .09 archive item ID (e.g., S01.11.09)
}

// NewUnarchiveItemCommand creates a new UnarchiveItemCommand
func NewUnarchiveItemCommand(repo ports.VaultRepository, archiveItemID string) *UnarchiveItemCommand {
	return &UnarchiveItemCommand{
		repo:          repo,
		ArchiveItemID: archiveItemID,
	}
}

// Validate checks if the unarchive operation is valid
func (c *UnarchiveItemCommand) Validate() error {
	if c.ArchiveItemID == "" {
		return &application.ValidationError{
			Field:   "itemID",
			Message: "item ID is required",
		}
	}

	if domain.ParseIDType(c.ArchiveItemID) != domain.IDTypeItem {
		return &application.ValidationError{
			Field:   "itemID",
			Message: fmt.Sprintf("expected item ID, got: %s", c.ArchiveItemID),
		}
	}

	if !domain.IsArchiveItem(c.ArchiveItemID) {
		return &application.ValidationError{
			Field:   "itemID",
			Message: fmt.Sprintf("%s is not an archive item (.09)", c.ArchiveItemID),
		}
	}

	return nil
}

// Execute runs the unarchive command
func (c *UnarchiveItemCommand) Execute(ctx context.Context) (*UnarchiveItemResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	// Infer destination category from archive item ID (S01.11.09 -> S01.11)
	dstCategoryID, err := domain.ParseCategory(c.ArchiveItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to determine destination: %w", err)
	}

	restoredItems, err := c.repo.UnarchiveItems(c.ArchiveItemID, dstCategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to unarchive: %w", err)
	}

	var names []string
	for _, item := range restoredItems {
		names = append(names, item.ID)
	}

	return &UnarchiveItemResult{
		ArchiveItemID: c.ArchiveItemID,
		RestoredItems: names,
		Message:       fmt.Sprintf("Restored %d items from %s", len(restoredItems), c.ArchiveItemID),
	}, nil
}

// UnarchiveEligibility contains the result of checking if a node can be unarchived
type UnarchiveEligibility struct {
	CanUnarchive bool
	Reason       string
}

// CheckUnarchiveEligibility determines if a node can be unarchived
func CheckUnarchiveEligibility(nodeID string, nodeType domain.IDType) UnarchiveEligibility {
	if nodeType != domain.IDTypeItem {
		return UnarchiveEligibility{
			CanUnarchive: false,
			Reason:       "only archive items (.09) can be unarchived",
		}
	}

	if !domain.IsArchiveItem(nodeID) {
		return UnarchiveEligibility{
			CanUnarchive: false,
			Reason:       "this item is not an archive (.09)",
		}
	}

	return UnarchiveEligibility{CanUnarchive: true}
}
