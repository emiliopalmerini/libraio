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

// ArchiveCategory returns the archive category ID for an area
func ArchiveCategory(area string) (string, error) {
	return domain.ArchiveCategory(area)
}
