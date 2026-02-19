package domain

import (
	"cmp"
	"slices"
)

// Scope represents a top-level scope in the vault (S00, S01, S02, S03)
type Scope struct {
	ID          string // e.g., "S00"
	Name        string // e.g., "System Management"
	Description string
	Path        string
}

// Area represents an area within a scope (e.g., S01.10-19 Lifestyle)
type Area struct {
	ID          string // e.g., "S01.10-19"
	Name        string // e.g., "Lifestyle"
	Description string
	Path        string
	ScopeID     string
}

// Category represents a category within an area (e.g., S01.11 Entertainment)
type Category struct {
	ID          string // e.g., "S01.11"
	Name        string // e.g., "Entertainment"
	Description string
	Path        string
	AreaID      string
	IsArchive   bool
}

// Item represents an individual item within a category (e.g., S01.11.11 Theatre)
type Item struct {
	ID          string // e.g., "S01.11.11"
	Name        string // e.g., "Theatre, 2025 Season"
	Description string
	Path        string
	CategoryID  string
}

// SearchResult represents a search match
type SearchResult struct {
	Type        IDType
	ID          string // Parent item ID (for navigation)
	Name        string // Filename
	Path        string // Full file path
	MatchedText string
}

// TreeNode represents a node in the vault tree for navigation
type TreeNode struct {
	Type       IDType
	ID         string
	Name       string
	Path       string
	Children   []*TreeNode
	IsExpanded bool
	Parent     *TreeNode
}

// Flatten returns all visible nodes in the tree (for list rendering)
func (n *TreeNode) Flatten() []*TreeNode {
	var result []*TreeNode
	n.flattenRecursive(&result, 0)
	return result
}

func (n *TreeNode) flattenRecursive(result *[]*TreeNode, depth int) {
	*result = append(*result, n)
	if n.IsExpanded {
		for _, child := range n.Children {
			child.flattenRecursive(result, depth+1)
		}
	}
}

// Depth returns the depth of this node in the tree
func (n *TreeNode) Depth() int {
	depth := 0
	current := n.Parent
	for current != nil {
		depth++
		current = current.Parent
	}
	return depth
}

// Toggle expands or collapses the node
func (n *TreeNode) Toggle() {
	n.IsExpanded = !n.IsExpanded
}

// Expand sets the node as expanded
func (n *TreeNode) Expand() {
	n.IsExpanded = true
}

// Collapse sets the node as collapsed
func (n *TreeNode) Collapse() {
	n.IsExpanded = false
}

// IDGetter defines the interface for entities that have an ID
type IDGetter interface {
	GetID() string
}

// GetID returns the ID of a Scope
func (s Scope) GetID() string { return s.ID }

// GetID returns the ID of an Area
func (a Area) GetID() string { return a.ID }

// GetID returns the ID of a Category
func (c Category) GetID() string { return c.ID }

// GetID returns the ID of an Item
func (i Item) GetID() string { return i.ID }

// SortByID sorts any slice of entities by their ID in ascending order.
// This generic function replaces the need for separate SortScopes, SortAreas,
// SortCategories, and SortItems functions.
func SortByID[T IDGetter](entities []T) {
	slices.SortFunc(entities, func(a, b T) int {
		return cmp.Compare(a.GetID(), b.GetID())
	})
}

// SortScopes sorts scopes by ID in ascending order
// Deprecated: Use SortByID[Scope](scopes) instead
func SortScopes(scopes []Scope) {
	SortByID(scopes)
}

// SortAreas sorts areas by ID in ascending order
// Deprecated: Use SortByID[Area](areas) instead
func SortAreas(areas []Area) {
	SortByID(areas)
}

// SortCategories sorts categories by ID in ascending order
// Deprecated: Use SortByID[Category](categories) instead
func SortCategories(categories []Category) {
	SortByID(categories)
}

// SortItems sorts items by ID in ascending order
// Deprecated: Use SortByID[Item](items) instead
func SortItems(items []Item) {
	SortByID(items)
}
