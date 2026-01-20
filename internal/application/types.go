package application

import "libraio/internal/domain"

// Re-export ID types for use by adapters
type IDType = domain.IDType

const (
	IDTypeUnknown  = domain.IDTypeUnknown
	IDTypeScope    = domain.IDTypeScope
	IDTypeArea     = domain.IDTypeArea
	IDTypeCategory = domain.IDTypeCategory
	IDTypeItem     = domain.IDTypeItem
	IDTypeFile     = domain.IDTypeFile
)

// Re-export domain types for use by adapters
type (
	TreeNode     = domain.TreeNode
	SearchResult = domain.SearchResult
	Scope        = domain.Scope
	Area         = domain.Area
	Category     = domain.Category
	Item         = domain.Item
)

// ParseIDType determines the type of a Johnny Decimal ID string
func ParseIDType(id string) IDType {
	return domain.ParseIDType(id)
}

// ParseArea extracts the area from a category or item ID
func ParseArea(id string) (string, error) {
	return domain.ParseArea(id)
}

// ParseCategory extracts the category from an item ID
func ParseCategory(id string) (string, error) {
	return domain.ParseCategory(id)
}

// ArchiveItemID returns the archive item ID (.09) for a category
func ArchiveItemID(categoryID string) (string, error) {
	return domain.ArchiveItemID(categoryID)
}

// IsArchiveItem checks if an item ID is an archive item (.09)
func IsArchiveItem(itemID string) bool {
	return domain.IsArchiveItem(itemID)
}

// GetIDHierarchy returns the full hierarchy of IDs leading to the given ID
func GetIDHierarchy(id string) []string {
	return domain.GetIDHierarchy(id)
}
