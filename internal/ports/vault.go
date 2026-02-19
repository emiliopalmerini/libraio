package ports

import "libraio/internal/domain"

// TreeReader provides read-only access to the vault tree structure
type TreeReader interface {
	BuildTree() (*domain.TreeNode, error)
	LoadChildren(node *domain.TreeNode) error
}

// PathResolver provides path resolution for vault entities
type PathResolver interface {
	GetPath(id string) (string, error)
	VaultPath() string
}

// VaultLister provides listing operations for vault entities
type VaultLister interface {
	ListScopes() ([]domain.Scope, error)
	ListAreas(scopeID string) ([]domain.Area, error)
	ListCategories(areaID string) ([]domain.Category, error)
	ListItems(categoryID string) ([]domain.Item, error)
}

// VaultSearcher provides search functionality
type VaultSearcher interface {
	Search(query string) ([]domain.SearchResult, error)
}

// VaultCreator provides creation operations
type VaultCreator interface {
	CreateScope(description string) (*domain.Scope, error)
	CreateArea(scopeID, description string) (*domain.Area, error)
	CreateCategory(areaID, description string) (*domain.Category, error)
	CreateItem(categoryID, description string) (*domain.Item, error)
}

// VaultMover provides move operations
type VaultMover interface {
	MoveItem(srcItemID, dstCategoryID string) (*domain.Item, error)
	MoveCategory(srcCategoryID, dstAreaID string) (*domain.Category, error)
}

// VaultArchiver provides archive operations
type VaultArchiver interface {
	ArchiveItem(srcItemID string) (*domain.Item, error)
	ArchiveCategory(srcCategoryID string) ([]*domain.Item, error)
}

// VaultUnarchiver provides unarchive operations
type VaultUnarchiver interface {
	UnarchiveItems(archiveItemID, dstCategoryID string) ([]*domain.Item, error)
}

// VaultRenamer provides rename operations
type VaultRenamer interface {
	RenameItem(itemID, newDescription string) (*domain.Item, error)
	RenameCategory(categoryID, newDescription string) (*domain.Category, error)
	RenameArea(areaID, newDescription string) (*domain.Area, error)
}

// VaultDeleter provides delete operations
type VaultDeleter interface {
	Delete(id string) error
}

// VaultRepository defines the full interface for vault storage operations.
// It composes all the smaller interfaces for backwards compatibility.
type VaultRepository interface {
	TreeReader
	PathResolver
	VaultLister
	VaultSearcher
	VaultCreator
	VaultMover
	VaultArchiver
	VaultUnarchiver
	VaultRenamer
	VaultDeleter
}
