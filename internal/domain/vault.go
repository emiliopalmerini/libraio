package domain

import "slices"

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
	JDexPath    string // Path to the JDex file (e.g., "S01.11.11 Theatre/S01.11.11 Theatre.md")
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

// SortScopes sorts scopes by ID in ascending order
func SortScopes(scopes []Scope) {
	slices.SortFunc(scopes, func(a, b Scope) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
}

// SortAreas sorts areas by ID in ascending order
func SortAreas(areas []Area) {
	slices.SortFunc(areas, func(a, b Area) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
}

// SortCategories sorts categories by ID in ascending order
func SortCategories(categories []Category) {
	slices.SortFunc(categories, func(a, b Category) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
}

// SortItems sorts items by ID in ascending order
func SortItems(items []Item) {
	slices.SortFunc(items, func(a, b Item) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
}
