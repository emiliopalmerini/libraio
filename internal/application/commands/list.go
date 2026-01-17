package commands

import (
	"context"

	"libraio/internal/domain"
	"libraio/internal/ports"
)

// ListScopesCommand lists all scopes in the vault
type ListScopesCommand struct {
	repo ports.VaultRepository
}

// NewListScopesCommand creates a new ListScopesCommand
func NewListScopesCommand(repo ports.VaultRepository) *ListScopesCommand {
	return &ListScopesCommand{repo: repo}
}

// Execute runs the list scopes command
func (c *ListScopesCommand) Execute(ctx context.Context) ([]domain.Scope, error) {
	return c.repo.ListScopes()
}

// ListAreasCommand lists all areas in a scope
type ListAreasCommand struct {
	repo    ports.VaultRepository
	ScopeID string
}

// NewListAreasCommand creates a new ListAreasCommand
func NewListAreasCommand(repo ports.VaultRepository, scopeID string) *ListAreasCommand {
	return &ListAreasCommand{
		repo:    repo,
		ScopeID: scopeID,
	}
}

// Execute runs the list areas command
func (c *ListAreasCommand) Execute(ctx context.Context) ([]domain.Area, error) {
	return c.repo.ListAreas(c.ScopeID)
}

// ListCategoriesCommand lists all categories in an area
type ListCategoriesCommand struct {
	repo   ports.VaultRepository
	AreaID string
}

// NewListCategoriesCommand creates a new ListCategoriesCommand
func NewListCategoriesCommand(repo ports.VaultRepository, areaID string) *ListCategoriesCommand {
	return &ListCategoriesCommand{
		repo:   repo,
		AreaID: areaID,
	}
}

// Execute runs the list categories command
func (c *ListCategoriesCommand) Execute(ctx context.Context) ([]domain.Category, error) {
	return c.repo.ListCategories(c.AreaID)
}

// ListItemsCommand lists all items in a category
type ListItemsCommand struct {
	repo       ports.VaultRepository
	CategoryID string
}

// NewListItemsCommand creates a new ListItemsCommand
func NewListItemsCommand(repo ports.VaultRepository, categoryID string) *ListItemsCommand {
	return &ListItemsCommand{
		repo:       repo,
		CategoryID: categoryID,
	}
}

// Execute runs the list items command
func (c *ListItemsCommand) Execute(ctx context.Context) ([]domain.Item, error) {
	return c.repo.ListItems(c.CategoryID)
}

// BuildTreeCommand builds the complete tree structure
type BuildTreeCommand struct {
	repo ports.VaultRepository
}

// NewBuildTreeCommand creates a new BuildTreeCommand
func NewBuildTreeCommand(repo ports.VaultRepository) *BuildTreeCommand {
	return &BuildTreeCommand{repo: repo}
}

// Execute runs the build tree command
func (c *BuildTreeCommand) Execute(ctx context.Context) (*domain.TreeNode, error) {
	return c.repo.BuildTree()
}
