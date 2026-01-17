package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"libraio/internal/domain"
)

// Repository implements ports.VaultRepository using the filesystem
type Repository struct {
	vaultPath string
}

// NewRepository creates a new filesystem repository
func NewRepository(vaultPath string) *Repository {
	// Expand ~ to home directory
	if strings.HasPrefix(vaultPath, "~") {
		home, _ := os.UserHomeDir()
		vaultPath = filepath.Join(home, vaultPath[1:])
	}
	return &Repository{vaultPath: vaultPath}
}

// VaultPath returns the expanded vault path
func (r *Repository) VaultPath() string {
	return r.vaultPath
}

// ListScopes returns all scopes in the vault
func (r *Repository) ListScopes() ([]domain.Scope, error) {
	entries, err := os.ReadDir(r.vaultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault: %w", err)
	}

	var scopes []domain.Scope
	scopeRegex := regexp.MustCompile(`^(S0[0-9]) (.+)$`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := scopeRegex.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		scopes = append(scopes, domain.Scope{
			ID:   matches[1],
			Name: matches[2],
			Path: filepath.Join(r.vaultPath, entry.Name()),
		})
	}

	sort.Slice(scopes, func(i, j int) bool {
		return scopes[i].ID < scopes[j].ID
	})

	return scopes, nil
}

// ListAreas returns all areas within a scope
func (r *Repository) ListAreas(scopeID string) ([]domain.Area, error) {
	scopePath, err := r.findScopePath(scopeID)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(scopePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scope: %w", err)
	}

	var areas []domain.Area
	areaRegex := regexp.MustCompile(`^(S0[0-9]\.[0-9]0-[0-9]9) (.+)$`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := areaRegex.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		areas = append(areas, domain.Area{
			ID:      matches[1],
			Name:    matches[2],
			Path:    filepath.Join(scopePath, entry.Name()),
			ScopeID: scopeID,
		})
	}

	sort.Slice(areas, func(i, j int) bool {
		return areas[i].ID < areas[j].ID
	})

	return areas, nil
}

// ListCategories returns all categories within an area
func (r *Repository) ListCategories(areaID string) ([]domain.Category, error) {
	areaPath, err := r.findAreaPath(areaID)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(areaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read area: %w", err)
	}

	var categories []domain.Category
	categoryRegex := regexp.MustCompile(`^(S0[0-9]\.[0-9][0-9]) (.+)$`)

	// Determine archive number for this area
	archiveID, _ := domain.ArchiveCategory(areaID)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := categoryRegex.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		categories = append(categories, domain.Category{
			ID:        matches[1],
			Name:      matches[2],
			Path:      filepath.Join(areaPath, entry.Name()),
			AreaID:    areaID,
			IsArchive: matches[1] == archiveID,
		})
	}

	sort.Slice(categories, func(i, j int) bool {
		return categories[i].ID < categories[j].ID
	})

	return categories, nil
}

// ListItems returns all items within a category
func (r *Repository) ListItems(categoryID string) ([]domain.Item, error) {
	categoryPath, err := r.findCategoryPath(categoryID)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(categoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read category: %w", err)
	}

	var items []domain.Item
	itemRegex := regexp.MustCompile(`^(S0[0-9]\.[0-9][0-9]\.[0-9][0-9]) (.+)$`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := itemRegex.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		itemPath := filepath.Join(categoryPath, entry.Name())
		folderName := entry.Name()
		items = append(items, domain.Item{
			ID:         matches[1],
			Name:       matches[2],
			Path:       itemPath,
			CategoryID: categoryID,
			JDexPath:   filepath.Join(itemPath, domain.JDexFileName(folderName)),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items, nil
}

// CreateCategory creates a new category in an area with standard zero items
func (r *Repository) CreateCategory(areaID, description string) (*domain.Category, error) {
	areaPath, err := r.findAreaPath(areaID)
	if err != nil {
		return nil, err
	}

	// Get existing categories to find next ID
	existing, err := r.ListCategories(areaID)
	if err != nil {
		return nil, err
	}

	var existingIDs []string
	for _, cat := range existing {
		existingIDs = append(existingIDs, cat.ID)
	}

	newID, err := domain.NextCategoryID(areaID, existingIDs)
	if err != nil {
		return nil, err
	}

	folderName := domain.FormatFolderName(newID, description)
	categoryPath := filepath.Join(areaPath, folderName)

	if err := os.MkdirAll(categoryPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	// Create standard zero items with rollback on failure
	if err := r.CreateStandardZeros(newID, categoryPath); err != nil {
		os.RemoveAll(categoryPath)
		return nil, fmt.Errorf("failed to create standard zeros: %w", err)
	}

	return &domain.Category{
		ID:     newID,
		Name:   description,
		Path:   categoryPath,
		AreaID: areaID,
	}, nil
}

// CreateStandardZeros creates all standard zero items in a category
func (r *Repository) CreateStandardZeros(categoryID, categoryPath string) error {
	for _, sz := range domain.StandardZeros {
		itemID := fmt.Sprintf("%s.%02d", categoryID, sz.Number)
		// Use context-aware naming for area-level categories
		itemName := domain.StandardZeroNameForContext(sz.Name, categoryID)
		folderName := domain.FormatFolderName(itemID, itemName)
		itemPath := filepath.Join(categoryPath, folderName)

		if err := os.MkdirAll(itemPath, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", itemName, err)
		}

		jdexPath := filepath.Join(itemPath, domain.JDexFileName(folderName))
		jdexContent := domain.StandardZeroReadmeTemplate(itemID, itemName, sz.Purpose)

		if err := os.WriteFile(jdexPath, []byte(jdexContent), 0644); err != nil {
			return fmt.Errorf("failed to create JDex for %s: %w", itemName, err)
		}
	}
	return nil
}

// CreateItem creates a new item in a category with a JDex file
func (r *Repository) CreateItem(categoryID, description string) (*domain.Item, error) {
	categoryPath, err := r.findCategoryPath(categoryID)
	if err != nil {
		return nil, err
	}

	// Get existing items to find next ID
	existing, err := r.ListItems(categoryID)
	if err != nil {
		return nil, err
	}

	var existingIDs []string
	for _, item := range existing {
		existingIDs = append(existingIDs, item.ID)
	}

	newID, err := domain.NextItemID(categoryID, existingIDs)
	if err != nil {
		return nil, err
	}

	folderName := domain.FormatFolderName(newID, description)
	itemPath := filepath.Join(categoryPath, folderName)

	if err := os.MkdirAll(itemPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	// Create JDex file
	jdexPath := filepath.Join(itemPath, domain.JDexFileName(folderName))
	jdexContent := domain.ReadmeTemplate(newID, description)

	if err := os.WriteFile(jdexPath, []byte(jdexContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create JDex: %w", err)
	}

	return &domain.Item{
		ID:         newID,
		Name:       description,
		Path:       itemPath,
		CategoryID: categoryID,
		JDexPath:   jdexPath,
	}, nil
}

// MoveItem moves an item to a different category
func (r *Repository) MoveItem(srcItemID, dstCategoryID string) (*domain.Item, error) {
	// Validate source is an item
	if domain.ParseIDType(srcItemID) != domain.IDTypeItem {
		return nil, fmt.Errorf("source must be an item, got: %s", srcItemID)
	}

	// Validate destination is a category
	if domain.ParseIDType(dstCategoryID) != domain.IDTypeCategory {
		return nil, fmt.Errorf("destination must be a category, got: %s", dstCategoryID)
	}

	// Check not moving to same category
	srcCategoryID, _ := domain.ParseCategory(srcItemID)
	if srcCategoryID == dstCategoryID {
		return nil, fmt.Errorf("item is already in category %s", dstCategoryID)
	}

	// Get source path and description
	srcPath, err := r.GetPath(srcItemID)
	if err != nil {
		return nil, fmt.Errorf("source item not found: %w", err)
	}
	description := domain.ExtractDescription(filepath.Base(srcPath))

	// Get destination category path
	dstCategoryPath, err := r.findCategoryPath(dstCategoryID)
	if err != nil {
		return nil, fmt.Errorf("destination category not found: %w", err)
	}

	// Get next available ID in destination
	existing, err := r.ListItems(dstCategoryID)
	if err != nil {
		return nil, err
	}
	var existingIDs []string
	for _, item := range existing {
		existingIDs = append(existingIDs, item.ID)
	}
	newID, err := domain.NextItemID(dstCategoryID, existingIDs)
	if err != nil {
		return nil, err
	}

	// Create new folder name and path
	newFolderName := domain.FormatFolderName(newID, description)
	dstPath := filepath.Join(dstCategoryPath, newFolderName)

	// Move the directory
	if err := os.Rename(srcPath, dstPath); err != nil {
		return nil, fmt.Errorf("failed to move item: %w", err)
	}

	// Find and update JDex file (check for new-style or legacy README.md)
	newJDexPath := filepath.Join(dstPath, domain.JDexFileName(newFolderName))
	oldFolderName := filepath.Base(srcPath)
	oldJDexPath := filepath.Join(dstPath, domain.JDexFileName(oldFolderName))
	legacyReadmePath := filepath.Join(dstPath, "README.md")

	// Try to find existing JDex file and update/rename it
	var sourcePath string
	if _, err := os.Stat(oldJDexPath); err == nil {
		sourcePath = oldJDexPath
	} else if _, err := os.Stat(legacyReadmePath); err == nil {
		sourcePath = legacyReadmePath
	}

	if sourcePath != "" {
		if content, err := os.ReadFile(sourcePath); err == nil {
			updated := domain.UpdateReadmeID(string(content), srcItemID, newID, description)
			// Write to new path
			if err := os.WriteFile(newJDexPath, []byte(updated), 0644); err == nil {
				// Remove old file if different from new
				if sourcePath != newJDexPath {
					os.Remove(sourcePath)
				}
			}
		}
	}

	return &domain.Item{
		ID:         newID,
		Name:       description,
		Path:       dstPath,
		CategoryID: dstCategoryID,
		JDexPath:   newJDexPath,
	}, nil
}

// MoveCategory moves a category to a different area
func (r *Repository) MoveCategory(srcCategoryID, dstAreaID string) (*domain.Category, error) {
	// Validate source is a category
	if domain.ParseIDType(srcCategoryID) != domain.IDTypeCategory {
		return nil, fmt.Errorf("source must be a category, got: %s", srcCategoryID)
	}

	// Validate destination is an area
	if domain.ParseIDType(dstAreaID) != domain.IDTypeArea {
		return nil, fmt.Errorf("destination must be an area, got: %s", dstAreaID)
	}

	// Check not moving to same area
	srcAreaID, _ := domain.ParseArea(srcCategoryID)
	if srcAreaID == dstAreaID {
		return nil, fmt.Errorf("category is already in area %s", dstAreaID)
	}

	// Get source path and description
	srcPath, err := r.GetPath(srcCategoryID)
	if err != nil {
		return nil, fmt.Errorf("source category not found: %w", err)
	}
	description := domain.ExtractDescription(filepath.Base(srcPath))

	// Get destination area path
	dstAreaPath, err := r.findAreaPath(dstAreaID)
	if err != nil {
		return nil, fmt.Errorf("destination area not found: %w", err)
	}

	// Get next available ID in destination
	existing, err := r.ListCategories(dstAreaID)
	if err != nil {
		return nil, err
	}
	var existingIDs []string
	for _, cat := range existing {
		existingIDs = append(existingIDs, cat.ID)
	}
	newID, err := domain.NextCategoryID(dstAreaID, existingIDs)
	if err != nil {
		return nil, err
	}

	// Create new folder name and path
	newFolderName := domain.FormatFolderName(newID, description)
	dstPath := filepath.Join(dstAreaPath, newFolderName)

	// Move the directory
	if err := os.Rename(srcPath, dstPath); err != nil {
		return nil, fmt.Errorf("failed to move category: %w", err)
	}

	// Update all item IDs within the category
	r.updateItemIDsInCategory(dstPath, srcCategoryID, newID)

	return &domain.Category{
		ID:     newID,
		Name:   description,
		Path:   dstPath,
		AreaID: dstAreaID,
	}, nil
}

// updateItemIDsInCategory updates all item IDs when a category is moved
func (r *Repository) updateItemIDsInCategory(categoryPath, _, newCategoryID string) {
	entries, err := os.ReadDir(categoryPath)
	if err != nil {
		return
	}

	itemRegex := regexp.MustCompile(`^(S0[0-9]\.[0-9][0-9]\.[0-9][0-9]) (.+)$`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := itemRegex.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		oldItemID := matches[1]
		description := matches[2]

		// Extract item number (last two digits)
		parts := strings.Split(oldItemID, ".")
		if len(parts) < 3 {
			continue
		}
		itemNum := parts[2]

		// Create new item ID
		newItemID := fmt.Sprintf("%s.%s", newCategoryID, itemNum)

		// Rename folder
		oldPath := filepath.Join(categoryPath, entry.Name())
		newFolderName := domain.FormatFolderName(newItemID, description)
		newPath := filepath.Join(categoryPath, newFolderName)

		if err := os.Rename(oldPath, newPath); err != nil {
			continue
		}

		// Find and update JDex file (check for new-style or legacy README.md)
		oldFolderName := entry.Name()
		newJDexPath := filepath.Join(newPath, domain.JDexFileName(newFolderName))
		oldJDexPath := filepath.Join(newPath, domain.JDexFileName(oldFolderName))
		legacyReadmePath := filepath.Join(newPath, "README.md")

		var sourcePath string
		if _, err := os.Stat(oldJDexPath); err == nil {
			sourcePath = oldJDexPath
		} else if _, err := os.Stat(legacyReadmePath); err == nil {
			sourcePath = legacyReadmePath
		}

		if sourcePath != "" {
			if content, err := os.ReadFile(sourcePath); err == nil {
				updated := domain.UpdateReadmeID(string(content), oldItemID, newItemID, description)
				if err := os.WriteFile(newJDexPath, []byte(updated), 0644); err == nil {
					if sourcePath != newJDexPath {
						os.Remove(sourcePath)
					}
				}
			}
		}
	}
}

// ArchiveItem moves an item to the archive category of its area
func (r *Repository) ArchiveItem(srcItemID string) (*domain.Item, error) {
	// Validate source is an item
	if domain.ParseIDType(srcItemID) != domain.IDTypeItem {
		return nil, fmt.Errorf("source must be an item, got: %s", srcItemID)
	}

	// Get the item's category and area
	srcCategoryID, err := domain.ParseCategory(srcItemID)
	if err != nil {
		return nil, err
	}

	areaID, err := domain.ParseArea(srcItemID)
	if err != nil {
		return nil, err
	}

	// Get archive category ID for this area
	archiveCategoryID, err := domain.ArchiveCategory(areaID)
	if err != nil {
		return nil, err
	}

	// Check if already in archive
	if srcCategoryID == archiveCategoryID {
		return nil, fmt.Errorf("item %s is already in archive category %s", srcItemID, archiveCategoryID)
	}

	// Verify archive category exists
	_, err = r.findCategoryPath(archiveCategoryID)
	if err != nil {
		return nil, fmt.Errorf("archive category %s not found: %w", archiveCategoryID, err)
	}

	// Get item description for link updates
	srcPath, err := r.GetPath(srcItemID)
	if err != nil {
		return nil, fmt.Errorf("source item not found: %w", err)
	}
	description := domain.ExtractDescription(filepath.Base(srcPath))

	// Move the item to archive category
	archivedItem, err := r.MoveItem(srcItemID, archiveCategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to move item to archive: %w", err)
	}

	// Update Obsidian links throughout the vault
	r.updateObsidianLinks(srcItemID, archivedItem.ID, description)

	return archivedItem, nil
}

// ArchiveCategory moves all non-standard-zero items to the archive category and deletes the source category
func (r *Repository) ArchiveCategory(srcCategoryID string) ([]*domain.Item, error) {
	// Validate source is a category
	if domain.ParseIDType(srcCategoryID) != domain.IDTypeCategory {
		return nil, fmt.Errorf("source must be a category, got: %s", srcCategoryID)
	}

	// Get area ID
	areaID, err := domain.ParseArea(srcCategoryID)
	if err != nil {
		return nil, err
	}

	// Get archive category ID
	archiveCategoryID, err := domain.ArchiveCategory(areaID)
	if err != nil {
		return nil, err
	}

	// Check if trying to archive the archive category itself
	if srcCategoryID == archiveCategoryID {
		return nil, fmt.Errorf("cannot archive the archive category %s", srcCategoryID)
	}

	// Verify archive category exists
	_, err = r.findCategoryPath(archiveCategoryID)
	if err != nil {
		return nil, fmt.Errorf("archive category %s not found: %w", archiveCategoryID, err)
	}

	// Get all items in the category
	items, err := r.ListItems(srcCategoryID)
	if err != nil {
		return nil, err
	}

	var archivedItems []*domain.Item

	// Archive each non-standard-zero item
	for _, item := range items {
		// Skip standard zeros (IDs .00-.09)
		itemNum, err := domain.ExtractNumber(item.ID)
		if err != nil {
			continue
		}
		if itemNum <= domain.StandardZeroMax {
			continue
		}

		// Archive this item
		archivedItem, err := r.ArchiveItem(item.ID)
		if err != nil {
			// Continue with other items even if one fails
			continue
		}
		archivedItems = append(archivedItems, archivedItem)
	}

	// Delete the source category (now empty of regular items)
	if err := r.Delete(srcCategoryID); err != nil {
		return archivedItems, fmt.Errorf("archived items but failed to delete category: %w", err)
	}

	return archivedItems, nil
}

// updateObsidianLinks updates all wiki links in the vault from oldID to newID
func (r *Repository) updateObsidianLinks(oldID, newID, description string) {
	// Walk the entire vault looking for markdown files
	filepath.Walk(r.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Only process markdown files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentStr := string(content)
		originalContent := contentStr

		// Replace various link formats:
		// [[S01.11.15 Theatre]] -> [[S01.19.11 Theatre]]
		// [[S01.11.15]] -> [[S01.19.11 Theatre]]
		// [[S01.11.15|Custom]] -> [[S01.19.11 Theatre|Custom]]
		// [[S01.11.15 Theatre|Custom]] -> [[S01.19.11 Theatre|Custom]]

		oldFullLink := fmt.Sprintf("[[%s %s]]", oldID, description)
		newFullLink := fmt.Sprintf("[[%s %s]]", newID, description)
		contentStr = strings.ReplaceAll(contentStr, oldFullLink, newFullLink)

		// Replace ID-only links: [[S01.11.15]] -> [[S01.19.11 Theatre]]
		oldIDLink := fmt.Sprintf("[[%s]]", oldID)
		contentStr = strings.ReplaceAll(contentStr, oldIDLink, newFullLink)

		// Replace aliased links: [[S01.11.15|...]] -> [[S01.19.11 Theatre|...]]
		// We need to use regex for this
		oldAliasPattern := regexp.MustCompile(`\[\[` + regexp.QuoteMeta(oldID) + `\|`)
		contentStr = oldAliasPattern.ReplaceAllString(contentStr, fmt.Sprintf("[[%s %s|", newID, description))

		// Replace full aliased links: [[S01.11.15 Theatre|...]] -> [[S01.19.11 Theatre|...]]
		oldFullAliasPattern := regexp.MustCompile(`\[\[` + regexp.QuoteMeta(oldID+" "+description) + `\|`)
		contentStr = oldFullAliasPattern.ReplaceAllString(contentStr, fmt.Sprintf("[[%s %s|", newID, description))

		// Only write if content changed
		if contentStr != originalContent {
			os.WriteFile(path, []byte(contentStr), 0644)
		}

		return nil
	})
}

// Delete removes an item, category, area, or scope by ID
func (r *Repository) Delete(id string) error {
	path, err := r.GetPath(id)
	if err != nil {
		return fmt.Errorf("not found: %w", err)
	}
	return os.RemoveAll(path)
}

// Search searches for items matching the query in folder names and filenames
func (r *Repository) Search(query string) ([]domain.SearchResult, error) {
	query = strings.ToLower(query)
	var results []domain.SearchResult
	seen := make(map[string]bool) // Avoid duplicates

	err := filepath.Walk(r.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Check directories (scopes, areas, categories)
		if info.IsDir() {
			name := info.Name()
			id := domain.ExtractID(name)
			idType := domain.ParseIDType(id)

			if idType == domain.IDTypeUnknown || seen[id] {
				return nil
			}

			// Match folder name
			if strings.Contains(strings.ToLower(name), query) {
				seen[id] = true
				results = append(results, domain.SearchResult{
					Type:        idType,
					ID:          id,
					Name:        domain.ExtractDescription(name),
					Path:        path,
					MatchedText: name,
				})
			}
			return nil
		}

		// Check all files for filename matches
		if !info.IsDir() {
			// Match against filename (without extension)
			filename := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			if !strings.Contains(strings.ToLower(filename), query) {
				return nil
			}

			// Find the nearest parent with a valid Johnny Decimal ID
			id, idPath := r.findNearestID(path)
			if id == "" || seen[id] {
				return nil
			}

			idType := domain.ParseIDType(id)
			seen[id] = true
			results = append(results, domain.SearchResult{
				Type:        idType,
				ID:          id,
				Name:        domain.ExtractDescription(filepath.Base(idPath)),
				Path:        idPath,
				MatchedText: filename, // Use matched filename for scoring
			})
		}

		return nil
	})

	return results, err
}

// findNearestID finds the nearest parent directory with a valid Johnny Decimal ID
func (r *Repository) findNearestID(path string) (string, string) {
	currentPath := filepath.Dir(path)

	for currentPath != r.vaultPath && currentPath != "/" && currentPath != "." {
		dirName := filepath.Base(currentPath)
		id := domain.ExtractID(dirName)
		idType := domain.ParseIDType(id)

		if idType != domain.IDTypeUnknown {
			return id, currentPath
		}

		currentPath = filepath.Dir(currentPath)
	}

	return "", ""
}

// BuildTree builds the complete vault tree
func (r *Repository) BuildTree() (*domain.TreeNode, error) {
	root := &domain.TreeNode{
		Type:       domain.IDTypeUnknown,
		ID:         "root",
		Name:       "Vault",
		Path:       r.vaultPath,
		IsExpanded: true,
	}

	scopes, err := r.ListScopes()
	if err != nil {
		return nil, err
	}

	for _, scope := range scopes {
		scopeNode := &domain.TreeNode{
			Type:   domain.IDTypeScope,
			ID:     scope.ID,
			Name:   scope.Name,
			Path:   scope.Path,
			Parent: root,
		}
		root.Children = append(root.Children, scopeNode)
	}

	return root, nil
}

// LoadChildren loads children for a node
func (r *Repository) LoadChildren(node *domain.TreeNode) error {
	if len(node.Children) > 0 {
		return nil // Already loaded
	}

	switch node.Type {
	case domain.IDTypeUnknown: // Root
		scopes, err := r.ListScopes()
		if err != nil {
			return err
		}
		for _, scope := range scopes {
			node.Children = append(node.Children, &domain.TreeNode{
				Type:   domain.IDTypeScope,
				ID:     scope.ID,
				Name:   scope.Name,
				Path:   scope.Path,
				Parent: node,
			})
		}

	case domain.IDTypeScope:
		areas, err := r.ListAreas(node.ID)
		if err != nil {
			return err
		}
		for _, area := range areas {
			node.Children = append(node.Children, &domain.TreeNode{
				Type:   domain.IDTypeArea,
				ID:     area.ID,
				Name:   area.Name,
				Path:   area.Path,
				Parent: node,
			})
		}

	case domain.IDTypeArea:
		categories, err := r.ListCategories(node.ID)
		if err != nil {
			return err
		}
		for _, cat := range categories {
			node.Children = append(node.Children, &domain.TreeNode{
				Type:   domain.IDTypeCategory,
				ID:     cat.ID,
				Name:   cat.Name,
				Path:   cat.Path,
				Parent: node,
			})
		}

	case domain.IDTypeCategory:
		items, err := r.ListItems(node.ID)
		if err != nil {
			return err
		}
		for _, item := range items {
			node.Children = append(node.Children, &domain.TreeNode{
				Type:   domain.IDTypeItem,
				ID:     item.ID,
				Name:   item.Name,
				Path:   item.Path,
				Parent: node,
			})
		}
	}

	return nil
}

// GetPath returns the filesystem path for an ID
func (r *Repository) GetPath(id string) (string, error) {
	idType := domain.ParseIDType(id)

	switch idType {
	case domain.IDTypeScope:
		return r.findScopePath(id)
	case domain.IDTypeArea:
		return r.findAreaPath(id)
	case domain.IDTypeCategory:
		return r.findCategoryPath(id)
	case domain.IDTypeItem:
		return r.findItemPath(id)
	default:
		return "", fmt.Errorf("unknown ID type: %s", id)
	}
}

// GetJDexPath returns the JDex file path for an item
func (r *Repository) GetJDexPath(itemID string) (string, error) {
	itemPath, err := r.findItemPath(itemID)
	if err != nil {
		return "", err
	}
	folderName := filepath.Base(itemPath)
	return filepath.Join(itemPath, domain.JDexFileName(folderName)), nil
}

// Helper methods for finding paths

func (r *Repository) findScopePath(scopeID string) (string, error) {
	entries, err := os.ReadDir(r.vaultPath)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), scopeID+" ") {
			return filepath.Join(r.vaultPath, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("scope not found: %s", scopeID)
}

func (r *Repository) findAreaPath(areaID string) (string, error) {
	scopeID, err := domain.ParseScope(areaID)
	if err != nil {
		return "", err
	}

	scopePath, err := r.findScopePath(scopeID)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(scopePath)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), areaID+" ") {
			return filepath.Join(scopePath, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("area not found: %s", areaID)
}

func (r *Repository) findCategoryPath(categoryID string) (string, error) {
	areaID, err := domain.ParseArea(categoryID)
	if err != nil {
		return "", err
	}

	areaPath, err := r.findAreaPath(areaID)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(areaPath)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), categoryID+" ") {
			return filepath.Join(areaPath, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("category not found: %s", categoryID)
}

func (r *Repository) findItemPath(itemID string) (string, error) {
	categoryID, err := domain.ParseCategory(itemID)
	if err != nil {
		return "", err
	}

	categoryPath, err := r.findCategoryPath(categoryID)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(categoryPath)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), itemID+" ") {
			return filepath.Join(categoryPath, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("item not found: %s", itemID)
}
