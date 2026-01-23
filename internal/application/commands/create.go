package commands

import (
	"context"
	"fmt"

	"libraio/internal/application"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

// CreateMode indicates what type of entity to create
type CreateMode int

const (
	CreateModeScope CreateMode = iota
	CreateModeArea
	CreateModeCategory
	CreateModeItem
)

// DetermineCreateMode returns what should be created based on the parent type
func DetermineCreateMode(parentType domain.IDType) (CreateMode, error) {
	switch parentType {
	case domain.IDTypeUnknown: // Root
		return CreateModeScope, nil
	case domain.IDTypeScope:
		return CreateModeArea, nil
	case domain.IDTypeArea:
		return CreateModeCategory, nil
	case domain.IDTypeCategory:
		return CreateModeItem, nil
	default:
		return 0, &application.ValidationError{
			Field:   "parentID",
			Message: fmt.Sprintf("cannot create under %s", parentType),
		}
	}
}

// CreateScopeResult contains the result of creating a scope
type CreateScopeResult struct {
	Scope   *domain.Scope
	Message string
}

// CreateScopeCommand creates a scope in the vault
type CreateScopeCommand struct {
	repo        ports.VaultRepository
	Description string
}

// NewCreateScopeCommand creates a new CreateScopeCommand
func NewCreateScopeCommand(repo ports.VaultRepository, description string) *CreateScopeCommand {
	return &CreateScopeCommand{
		repo:        repo,
		Description: description,
	}
}

// Validate checks if the create operation is valid
func (c *CreateScopeCommand) Validate() error {
	if c.Description == "" {
		return &application.ValidationError{
			Field:   "description",
			Message: "description is required",
		}
	}
	return nil
}

// Execute runs the create scope command
func (c *CreateScopeCommand) Execute(ctx context.Context) (*CreateScopeResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	scope, err := c.repo.CreateScope(c.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create scope: %w", err)
	}

	return &CreateScopeResult{
		Scope:   scope,
		Message: fmt.Sprintf("Created scope: %s %s", scope.ID, scope.Name),
	}, nil
}

// CreateAreaResult contains the result of creating an area
type CreateAreaResult struct {
	Area    *domain.Area
	Message string
}

// CreateAreaCommand creates an area in a scope
type CreateAreaCommand struct {
	repo        ports.VaultRepository
	ScopeID     string
	Description string
}

// NewCreateAreaCommand creates a new CreateAreaCommand
func NewCreateAreaCommand(repo ports.VaultRepository, scopeID, description string) *CreateAreaCommand {
	return &CreateAreaCommand{
		repo:        repo,
		ScopeID:     scopeID,
		Description: description,
	}
}

// Validate checks if the create operation is valid
func (c *CreateAreaCommand) Validate() error {
	if c.ScopeID == "" {
		return &application.ValidationError{
			Field:   "scopeID",
			Message: "scope ID is required",
		}
	}

	if c.Description == "" {
		return &application.ValidationError{
			Field:   "description",
			Message: "description is required",
		}
	}

	if domain.ParseIDType(c.ScopeID) != domain.IDTypeScope {
		return &application.ValidationError{
			Field:   "scopeID",
			Message: fmt.Sprintf("expected scope ID, got: %s", c.ScopeID),
		}
	}

	return nil
}

// Execute runs the create area command
func (c *CreateAreaCommand) Execute(ctx context.Context) (*CreateAreaResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	area, err := c.repo.CreateArea(c.ScopeID, c.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create area: %w", err)
	}

	return &CreateAreaResult{
		Area:    area,
		Message: fmt.Sprintf("Created area: %s %s", area.ID, area.Name),
	}, nil
}

// CreateItemResult contains the result of creating an item
type CreateItemResult struct {
	Item    *domain.Item
	Message string
}

// CreateItemCommand creates an item in a category
type CreateItemCommand struct {
	repo        ports.VaultRepository
	CategoryID  string
	Description string
}

// NewCreateItemCommand creates a new CreateItemCommand
func NewCreateItemCommand(repo ports.VaultRepository, categoryID, description string) *CreateItemCommand {
	return &CreateItemCommand{
		repo:        repo,
		CategoryID:  categoryID,
		Description: description,
	}
}

// Validate checks if the create operation is valid
func (c *CreateItemCommand) Validate() error {
	if c.CategoryID == "" {
		return &application.ValidationError{
			Field:   "categoryID",
			Message: "category ID is required",
		}
	}

	if c.Description == "" {
		return &application.ValidationError{
			Field:   "description",
			Message: "description is required",
		}
	}

	if domain.ParseIDType(c.CategoryID) != domain.IDTypeCategory {
		return &application.ValidationError{
			Field:   "categoryID",
			Message: fmt.Sprintf("expected category ID, got: %s", c.CategoryID),
		}
	}

	return nil
}

// Execute runs the create item command
func (c *CreateItemCommand) Execute(ctx context.Context) (*CreateItemResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	item, err := c.repo.CreateItem(c.CategoryID, c.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	return &CreateItemResult{
		Item:    item,
		Message: fmt.Sprintf("Created item: %s %s", item.ID, item.Name),
	}, nil
}

// CreateCategoryResult contains the result of creating a category
type CreateCategoryResult struct {
	Category *domain.Category
	Message  string
}

// CreateCategoryCommand creates a category in an area
type CreateCategoryCommand struct {
	repo        ports.VaultRepository
	AreaID      string
	Description string
}

// NewCreateCategoryCommand creates a new CreateCategoryCommand
func NewCreateCategoryCommand(repo ports.VaultRepository, areaID, description string) *CreateCategoryCommand {
	return &CreateCategoryCommand{
		repo:        repo,
		AreaID:      areaID,
		Description: description,
	}
}

// Validate checks if the create operation is valid
func (c *CreateCategoryCommand) Validate() error {
	if c.AreaID == "" {
		return &application.ValidationError{
			Field:   "areaID",
			Message: "area ID is required",
		}
	}

	if c.Description == "" {
		return &application.ValidationError{
			Field:   "description",
			Message: "description is required",
		}
	}

	if domain.ParseIDType(c.AreaID) != domain.IDTypeArea {
		return &application.ValidationError{
			Field:   "areaID",
			Message: fmt.Sprintf("expected area ID, got: %s", c.AreaID),
		}
	}

	return nil
}

// Execute runs the create category command
func (c *CreateCategoryCommand) Execute(ctx context.Context) (*CreateCategoryResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	cat, err := c.repo.CreateCategory(c.AreaID, c.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return &CreateCategoryResult{
		Category: cat,
		Message:  fmt.Sprintf("Created category: %s %s", cat.ID, cat.Name),
	}, nil
}
