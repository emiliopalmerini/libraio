package ports

import "libraio/internal/domain"

// VaultRepository defines the interface for vault storage operations
type VaultRepository interface {
	// List operations
	ListScopes() ([]domain.Scope, error)
	ListAreas(scopeID string) ([]domain.Area, error)
	ListCategories(areaID string) ([]domain.Category, error)
	ListItems(categoryID string) ([]domain.Item, error)

	// Create operations
	CreateCategory(areaID, description string) (*domain.Category, error)
	CreateItem(categoryID, description string) (*domain.Item, error)

	// Move operations
	MoveItem(srcItemID, dstCategoryID string) (*domain.Item, error)
	MoveCategory(srcCategoryID, dstAreaID string) (*domain.Category, error)

	// Delete operations
	Delete(id string) error

	// Search
	Search(query string) ([]domain.SearchResult, error)

	// Tree operations
	BuildTree() (*domain.TreeNode, error)
	LoadChildren(node *domain.TreeNode) error

	// Path resolution
	GetPath(id string) (string, error)
	GetReadmePath(itemID string) (string, error)
}
