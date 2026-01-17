package commands

import (
	"context"
	"fmt"
	"strings"

	"libraio/internal/application"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// ArchiveItemResult contains the result of archiving an item
type ArchiveItemResult struct {
	OriginalID   string
	ArchivedItem *domain.Item
	Message      string
}

// ArchiveItemCommand archives a single item to its area's archive category
type ArchiveItemCommand struct {
	repo   ports.VaultRepository
	ItemID string
}

// NewArchiveItemCommand creates a new ArchiveItemCommand
func NewArchiveItemCommand(repo ports.VaultRepository, itemID string) *ArchiveItemCommand {
	return &ArchiveItemCommand{
		repo:   repo,
		ItemID: itemID,
	}
}

// Validate checks if the item can be archived
func (c *ArchiveItemCommand) Validate() error {
	if c.ItemID == "" {
		return &application.ValidationError{
			Field:   "itemID",
			Message: "item ID is required",
		}
	}

	if domain.ParseIDType(c.ItemID) != domain.IDTypeItem {
		return &application.ValidationError{
			Field:   "itemID",
			Message: fmt.Sprintf("expected item ID, got: %s", c.ItemID),
		}
	}

	categoryID, err := domain.ParseCategory(c.ItemID)
	if err != nil {
		return &application.ValidationError{
			Field:   "itemID",
			Message: fmt.Sprintf("cannot parse category from item: %v", err),
		}
	}

	// Archive categories end with "9" (e.g., S01.19, S01.29)
	if strings.HasSuffix(categoryID, "9") {
		return &application.ArchiveError{
			ID:     c.ItemID,
			Reason: "item is already in an archive category",
		}
	}

	return nil
}

// Execute runs the archive command
func (c *ArchiveItemCommand) Execute(ctx context.Context) (*ArchiveItemResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	archivedItem, err := c.repo.ArchiveItem(c.ItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to archive item: %w", err)
	}

	return &ArchiveItemResult{
		OriginalID:   c.ItemID,
		ArchivedItem: archivedItem,
		Message:      fmt.Sprintf("Archived %s -> %s", c.ItemID, archivedItem.ID),
	}, nil
}

// ArchiveCategoryResult contains the result of archiving a category
type ArchiveCategoryResult struct {
	OriginalCategoryID string
	ArchivedItems      []*domain.Item
	Message            string
}

// ArchiveCategoryCommand archives all items in a category to the archive
type ArchiveCategoryCommand struct {
	repo       ports.VaultRepository
	CategoryID string
}

// NewArchiveCategoryCommand creates a new ArchiveCategoryCommand
func NewArchiveCategoryCommand(repo ports.VaultRepository, categoryID string) *ArchiveCategoryCommand {
	return &ArchiveCategoryCommand{
		repo:       repo,
		CategoryID: categoryID,
	}
}

// Validate checks if the category can be archived
func (c *ArchiveCategoryCommand) Validate() error {
	if c.CategoryID == "" {
		return &application.ValidationError{
			Field:   "categoryID",
			Message: "category ID is required",
		}
	}

	if domain.ParseIDType(c.CategoryID) != domain.IDTypeCategory {
		return &application.ValidationError{
			Field:   "categoryID",
			Message: fmt.Sprintf("expected category ID, got: %s", c.CategoryID),
		}
	}

	// Cannot archive the archive category itself (ends with "9")
	if strings.HasSuffix(c.CategoryID, "9") {
		return &application.ArchiveError{
			ID:     c.CategoryID,
			Reason: "cannot archive the archive category",
		}
	}

	return nil
}

// Execute runs the archive category command
func (c *ArchiveCategoryCommand) Execute(ctx context.Context) (*ArchiveCategoryResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	archivedItems, err := c.repo.ArchiveCategory(c.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to archive category: %w", err)
	}

	return &ArchiveCategoryResult{
		OriginalCategoryID: c.CategoryID,
		ArchivedItems:      archivedItems,
		Message:            fmt.Sprintf("Archived %d items from %s", len(archivedItems), c.CategoryID),
	}, nil
}

// ArchiveEligibility contains the result of checking if a node can be archived
type ArchiveEligibility struct {
	CanArchive bool
	Reason     string
}

// CheckArchiveEligibility determines if a node can be archived
func CheckArchiveEligibility(nodeID string, nodeType domain.IDType) ArchiveEligibility {
	switch nodeType {
	case domain.IDTypeItem:
		cmd := &ArchiveItemCommand{ItemID: nodeID}
		if err := cmd.Validate(); err != nil {
			return ArchiveEligibility{CanArchive: false, Reason: err.Error()}
		}
		return ArchiveEligibility{CanArchive: true}

	case domain.IDTypeCategory:
		cmd := &ArchiveCategoryCommand{CategoryID: nodeID}
		if err := cmd.Validate(); err != nil {
			return ArchiveEligibility{CanArchive: false, Reason: err.Error()}
		}
		return ArchiveEligibility{CanArchive: true}

	default:
		return ArchiveEligibility{
			CanArchive: false,
			Reason:     fmt.Sprintf("cannot archive %s (only items and categories can be archived)", nodeType),
		}
	}
}
