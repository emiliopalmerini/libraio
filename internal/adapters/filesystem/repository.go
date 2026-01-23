package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"libraio/internal/domain"
	"libraio/internal/ports"
)

// Repository implements ports.VaultRepository using the filesystem
type Repository struct {
	vaultPath string
	index     ports.VaultIndex // Optional cache for faster operations
}

// RepoOption is a functional option for configuring Repository
type RepoOption func(*Repository)

// WithIndex enables SQLite caching for the repository
func WithIndex(index ports.VaultIndex) RepoOption {
	return func(r *Repository) {
		r.index = index
	}
}

// LinkReplacement defines a single link transformation for vault-wide updates
type LinkReplacement struct {
	Old     string // Pattern to find (literal string or regex if IsRegex is true)
	New     string // Replacement string
	IsRegex bool   // If true, Old is treated as a regex pattern
}

// updateJDexFile finds and updates a JDex file (or legacy README.md) in the given directory.
// It looks for the old JDex file or README.md, applies the transform function, writes to newJDexPath,
// and removes the old file if different from the new path.
// Returns nil if no JDex file exists (not an error condition).
func updateJDexFile(dstPath, oldFolderName, newJDexPath string, transform func(string) string) error {
	oldJDexPath := filepath.Join(dstPath, domain.JDexFileName(oldFolderName))
	legacyReadmePath := filepath.Join(dstPath, "README.md")

	var sourcePath string
	if _, err := os.Stat(oldJDexPath); err == nil {
		sourcePath = oldJDexPath
	} else if _, err := os.Stat(legacyReadmePath); err == nil {
		sourcePath = legacyReadmePath
	}

	if sourcePath == "" {
		return nil // No JDex file to update
	}

	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read JDex file %s: %w", sourcePath, err)
	}

	updated := transform(string(content))
	if err := os.WriteFile(newJDexPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write JDex file %s: %w", newJDexPath, err)
	}

	if sourcePath != newJDexPath {
		if err := os.Remove(sourcePath); err != nil {
			return fmt.Errorf("failed to remove old JDex file %s: %w", sourcePath, err)
		}
	}
	return nil
}

// nextAvailableItemID returns the next available item ID in a category
func (r *Repository) nextAvailableItemID(categoryID string) (string, error) {
	existing, err := r.ListItems(categoryID)
	if err != nil {
		return "", err
	}

	var existingIDs []string
	for _, item := range existing {
		existingIDs = append(existingIDs, item.ID)
	}

	return domain.NextItemID(categoryID, existingIDs)
}

// nextAvailableCategoryID returns the next available category ID in an area
func (r *Repository) nextAvailableCategoryID(areaID string) (string, error) {
	existing, err := r.ListCategories(areaID)
	if err != nil {
		return "", err
	}

	var existingIDs []string
	for _, cat := range existing {
		existingIDs = append(existingIDs, cat.ID)
	}

	return domain.NextCategoryID(areaID, existingIDs)
}

// nextAvailableScopeID returns the next available scope ID
func (r *Repository) nextAvailableScopeID() (string, error) {
	existing, err := r.ListScopes()
	if err != nil {
		return "", err
	}

	var existingIDs []string
	for _, scope := range existing {
		existingIDs = append(existingIDs, scope.ID)
	}

	return domain.NextScopeID(existingIDs)
}

// nextAvailableAreaID returns the next available area ID in a scope
func (r *Repository) nextAvailableAreaID(scopeID string) (string, error) {
	existing, err := r.ListAreas(scopeID)
	if err != nil {
		return "", err
	}

	var existingIDs []string
	for _, area := range existing {
		existingIDs = append(existingIDs, area.ID)
	}

	return domain.NextAreaID(scopeID, existingIDs)
}

// NewRepository creates a new filesystem repository
func NewRepository(vaultPath string, opts ...RepoOption) *Repository {
	// Expand ~ to home directory
	if strings.HasPrefix(vaultPath, "~") {
		home, _ := os.UserHomeDir()
		vaultPath = filepath.Join(home, vaultPath[1:])
	}
	r := &Repository{vaultPath: vaultPath}
	for _, opt := range opts {
		opt(r)
	}
	return r
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

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := domain.ScopeFolderRegex.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		scopes = append(scopes, domain.Scope{
			ID:   matches[1],
			Name: matches[2],
			Path: filepath.Join(r.vaultPath, entry.Name()),
		})
	}

	domain.SortScopes(scopes)

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

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := domain.AreaFolderRegex.FindStringSubmatch(entry.Name())
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

	domain.SortAreas(areas)

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

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := domain.CategoryFolderRegex.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		categories = append(categories, domain.Category{
			ID:     matches[1],
			Name:   matches[2],
			Path:   filepath.Join(areaPath, entry.Name()),
			AreaID: areaID,
		})
	}

	domain.SortCategories(categories)

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

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := domain.ItemFolderRegex.FindStringSubmatch(entry.Name())
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

	domain.SortItems(items)

	return items, nil
}

// CreateScope creates a new scope in the vault
func (r *Repository) CreateScope(description string) (*domain.Scope, error) {
	newID, err := r.nextAvailableScopeID()
	if err != nil {
		return nil, err
	}

	folderName := domain.FormatFolderName(newID, description)
	scopePath := filepath.Join(r.vaultPath, folderName)

	if err := os.MkdirAll(scopePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create scope: %w", err)
	}

	return &domain.Scope{
		ID:   newID,
		Name: description,
		Path: scopePath,
	}, nil
}

// CreateArea creates a new area in a scope
func (r *Repository) CreateArea(scopeID, description string) (*domain.Area, error) {
	scopePath, err := r.findScopePath(scopeID)
	if err != nil {
		return nil, err
	}

	newID, err := r.nextAvailableAreaID(scopeID)
	if err != nil {
		return nil, err
	}

	folderName := domain.FormatFolderName(newID, description)
	areaPath := filepath.Join(scopePath, folderName)

	if err := os.MkdirAll(areaPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create area: %w", err)
	}

	return &domain.Area{
		ID:      newID,
		Name:    description,
		Path:    areaPath,
		ScopeID: scopeID,
	}, nil
}

// CreateCategory creates a new category in an area with standard zero items
func (r *Repository) CreateCategory(areaID, description string) (*domain.Category, error) {
	areaPath, err := r.findAreaPath(areaID)
	if err != nil {
		return nil, err
	}

	newID, err := r.nextAvailableCategoryID(areaID)
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

	newID, err := r.nextAvailableItemID(categoryID)
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

	newID, err := r.nextAvailableItemID(dstCategoryID)
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

	// Update JDex file (best-effort, non-critical)
	oldFolderName := filepath.Base(srcPath)
	newJDexPath := filepath.Join(dstPath, domain.JDexFileName(newFolderName))
	_ = updateJDexFile(dstPath, oldFolderName, newJDexPath, func(content string) string {
		return domain.UpdateReadmeID(content, srcItemID, newID, description)
	})

	// Update Obsidian links throughout the vault
	r.updateObsidianLinksWithCache(srcItemID, newID, description)

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

	newID, err := r.nextAvailableCategoryID(dstAreaID)
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

	// Update all item IDs within the category (also updates Obsidian links)
	r.updateItemIDsInCategory(dstPath, srcCategoryID, newID)

	// Update links to the category itself
	r.updateObsidianLinksWithCache(srcCategoryID, newID, description)

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

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := domain.ItemFolderRegex.FindStringSubmatch(entry.Name())
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

		// Update JDex file (best-effort, non-critical)
		oldFolderName := entry.Name()
		newJDexPath := filepath.Join(newPath, domain.JDexFileName(newFolderName))
		_ = updateJDexFile(newPath, oldFolderName, newJDexPath, func(content string) string {
			return domain.UpdateReadmeID(content, oldItemID, newItemID, description)
		})

		// Update Obsidian links for this item
		r.updateObsidianLinksWithCache(oldItemID, newItemID, description)
	}
}

// ArchiveItem moves an item to the category's .09 Archive folder
func (r *Repository) ArchiveItem(srcItemID string) (*domain.Item, error) {
	// Validate source is an item
	if domain.ParseIDType(srcItemID) != domain.IDTypeItem {
		return nil, fmt.Errorf("source must be an item, got: %s", srcItemID)
	}

	// Check if already an archive item
	if domain.IsArchiveItem(srcItemID) {
		return nil, fmt.Errorf("item %s is already an archive item", srcItemID)
	}

	// Get the item's category
	srcCategoryID, err := domain.ParseCategory(srcItemID)
	if err != nil {
		return nil, err
	}

	// Get archive item ID for this category (.09)
	archiveItemID, err := domain.ArchiveItemID(srcCategoryID)
	if err != nil {
		return nil, err
	}

	// Get source path
	srcPath, err := r.GetPath(srcItemID)
	if err != nil {
		return nil, fmt.Errorf("source item not found: %w", err)
	}
	description := domain.ExtractDescription(filepath.Base(srcPath))
	folderName := filepath.Base(srcPath)

	// Get archive item path
	archivePath, err := r.findItemPath(archiveItemID)
	if err != nil {
		return nil, fmt.Errorf("archive item %s not found: %w", archiveItemID, err)
	}

	// Archived items lose their ID - folder is renamed with [Archived] prefix
	archivedFolderName := "[Archived] " + description
	dstPath := filepath.Join(archivePath, archivedFolderName)
	if err := os.Rename(srcPath, dstPath); err != nil {
		return nil, fmt.Errorf("failed to move item to archive: %w", err)
	}

	// Update JDex file (rename from "S01.11.15 Theatre.md" to "[Archived] Theatre.md", best-effort)
	newJDexPath := filepath.Join(dstPath, archivedFolderName+".md")
	_ = updateJDexFile(dstPath, folderName, newJDexPath, func(content string) string {
		return domain.UpdateReadmeForArchive(content, srcItemID, description)
	})

	// Update Obsidian links throughout the vault
	r.updateObsidianLinksForArchive(srcItemID, description)

	// Return the archived item (ID is now empty since it's archived)
	return &domain.Item{
		ID:         "", // No ID after archiving
		Name:       description,
		Path:       dstPath,
		CategoryID: srcCategoryID,
		JDexPath:   newJDexPath,
	}, nil
}

// ArchiveCategory moves all non-standard-zero items to the category's .09 Archive folder
func (r *Repository) ArchiveCategory(srcCategoryID string) ([]*domain.Item, error) {
	// Validate source is a category
	if domain.ParseIDType(srcCategoryID) != domain.IDTypeCategory {
		return nil, fmt.Errorf("source must be a category, got: %s", srcCategoryID)
	}

	// Get archive item ID (.09) for this category
	archiveItemID, err := domain.ArchiveItemID(srcCategoryID)
	if err != nil {
		return nil, err
	}

	// Verify archive item exists
	_, err = r.findItemPath(archiveItemID)
	if err != nil {
		return nil, fmt.Errorf("archive item %s not found: %w", archiveItemID, err)
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

	return archivedItems, nil
}

// ArchiveCategoryToArea moves a category to the area's .X0.09 Archive folder
func (r *Repository) ArchiveCategoryToArea(srcCategoryID string) (*domain.Category, error) {
	// Validate source is a category
	if domain.ParseIDType(srcCategoryID) != domain.IDTypeCategory {
		return nil, fmt.Errorf("source must be a category, got: %s", srcCategoryID)
	}

	// Management categories can't be archived
	if domain.IsAreaManagementCategory(srcCategoryID) {
		return nil, fmt.Errorf("cannot archive management category %s", srcCategoryID)
	}

	// Get the area archive item ID (.X0.09)
	areaArchiveItemID, err := domain.AreaArchiveItemID(srcCategoryID)
	if err != nil {
		return nil, err
	}

	// Get source category path
	srcPath, err := r.GetPath(srcCategoryID)
	if err != nil {
		return nil, fmt.Errorf("source category not found: %w", err)
	}
	description := domain.ExtractDescription(filepath.Base(srcPath))
	folderName := filepath.Base(srcPath)

	// Get area archive item path
	archivePath, err := r.findItemPath(areaArchiveItemID)
	if err != nil {
		return nil, fmt.Errorf("area archive item %s not found: %w", areaArchiveItemID, err)
	}

	// Move the category folder into the area archive folder
	dstPath := filepath.Join(archivePath, folderName)
	if err := os.Rename(srcPath, dstPath); err != nil {
		return nil, fmt.Errorf("failed to move category to area archive: %w", err)
	}

	// Update Obsidian links throughout the vault
	r.updateObsidianLinks(srcCategoryID, srcCategoryID, description)

	return &domain.Category{
		ID:     srcCategoryID,
		Name:   description,
		Path:   dstPath,
		AreaID: "", // No longer has a direct area parent
	}, nil
}

// updateVaultLinks walks the vault and applies link replacements to all markdown files
func (r *Repository) updateVaultLinks(replacements []LinkReplacement) {
	filepath.Walk(r.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentStr := string(content)
		updated := applyLinkReplacements(contentStr, replacements)

		if updated != contentStr {
			_ = os.WriteFile(path, []byte(updated), 0644) // Best-effort write
		}

		return nil
	})
}

// buildLinkReplacements creates the standard set of wiki link replacements for renaming/moving items.
// It handles all Obsidian link formats: [[ID Name]], [[ID]], [[ID|Alias]], [[ID Name|Alias]]
func buildLinkReplacements(oldID, description, newLinkText, newAliasPrefix string) []LinkReplacement {
	return []LinkReplacement{
		// [[S01.11.15 Theatre]] -> new link
		{Old: fmt.Sprintf("[[%s %s]]", oldID, description), New: newLinkText},
		// [[S01.11.15]] -> new link
		{Old: fmt.Sprintf("[[%s]]", oldID), New: newLinkText},
		// [[S01.11.15|Custom]] -> [[new|Custom]]
		{Old: `\[\[` + regexp.QuoteMeta(oldID) + `\|`, New: newAliasPrefix, IsRegex: true},
		// [[S01.11.15 Theatre|Custom]] -> [[new|Custom]]
		{Old: `\[\[` + regexp.QuoteMeta(oldID+" "+description) + `\|`, New: newAliasPrefix, IsRegex: true},
	}
}

// applyLinkReplacements applies a set of link replacements to content
func applyLinkReplacements(content string, replacements []LinkReplacement) string {
	for _, repl := range replacements {
		if repl.IsRegex {
			re := regexp.MustCompile(repl.Old)
			content = re.ReplaceAllString(content, repl.New)
		} else {
			content = strings.ReplaceAll(content, repl.Old, repl.New)
		}
	}
	return content
}

// updateObsidianLinksForArchive updates all wiki links when archiving (adds [Archived] prefix)
// e.g., [[S01.11.15 Theatre]] -> [[[Archived] Theatre]], [[S01.11.15]] -> [[[Archived] Theatre]]
func (r *Repository) updateObsidianLinksForArchive(oldID, description string) {
	archivedName := "[Archived] " + description
	newLink := fmt.Sprintf("[[%s]]", archivedName)
	newAliasPrefix := fmt.Sprintf("[[%s|", archivedName)

	r.updateVaultLinks(buildLinkReplacements(oldID, description, newLink, newAliasPrefix))
}

// updateObsidianLinksWithCache updates wiki links using the index if available
func (r *Repository) updateObsidianLinksWithCache(oldID, newID, description string) {
	newFullLink := fmt.Sprintf("[[%s %s]]", newID, description)
	newAliasPrefix := fmt.Sprintf("[[%s %s|", newID, description)
	replacements := buildLinkReplacements(oldID, description, newFullLink, newAliasPrefix)

	if r.index != nil {
		// Use indexed lookup for O(k) performance where k = files with links
		edges, err := r.index.FindLinksToID(oldID)
		if err == nil && len(edges) > 0 {
			for _, edge := range edges {
				fullPath := filepath.Join(r.vaultPath, edge.SourcePath)
				content, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}

				contentStr := string(content)
				updated := applyLinkReplacements(contentStr, replacements)

				if updated != contentStr {
					_ = os.WriteFile(fullPath, []byte(updated), 0644) // Best-effort write
				}
			}

			// Update edge targets in the index
			if tx, err := r.index.BeginTx(); err == nil {
				tx.UpdateEdgeTarget(oldID, newID, newFullLink)
				tx.Commit()
			}
			return
		}
	}

	// Fallback: full vault walk
	r.updateObsidianLinks(oldID, newID, description)
}

// updateObsidianLinks updates all wiki links in the vault from oldID to newID
func (r *Repository) updateObsidianLinks(oldID, newID, description string) {
	newFullLink := fmt.Sprintf("[[%s %s]]", newID, description)
	newAliasPrefix := fmt.Sprintf("[[%s %s|", newID, description)

	r.updateVaultLinks(buildLinkReplacements(oldID, description, newFullLink, newAliasPrefix))
}

// Delete removes an item, category, area, or scope by ID
func (r *Repository) Delete(id string) error {
	path, err := r.GetPath(id)
	if err != nil {
		return fmt.Errorf("not found: %w", err)
	}
	return os.RemoveAll(path)
}

// Search searches for files matching the query
func (r *Repository) Search(query string) ([]domain.SearchResult, error) {
	query = strings.ToLower(query)
	var results []domain.SearchResult
	seen := make(map[string]bool) // Avoid duplicate file paths

	err := filepath.Walk(r.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Only search files
		if info.IsDir() {
			return nil
		}

		// Match against filename (with or without extension)
		filename := info.Name()
		filenameNoExt := strings.TrimSuffix(filename, filepath.Ext(filename))
		if !strings.Contains(strings.ToLower(filename), query) &&
			!strings.Contains(strings.ToLower(filenameNoExt), query) {
			return nil
		}

		// Avoid duplicates
		if seen[path] {
			return nil
		}
		seen[path] = true

		// Find the nearest parent with a valid Johnny Decimal ID (item)
		id, _ := r.findNearestID(path)

		results = append(results, domain.SearchResult{
			Type:        domain.IDTypeFile,
			ID:          id, // Parent item ID for navigation
			Name:        filename,
			Path:        path,
			MatchedText: filenameNoExt,
		})

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

	case domain.IDTypeItem:
		// Load files inside item directory
		entries, err := os.ReadDir(node.Path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue // Skip subdirectories
			}
			node.Children = append(node.Children, &domain.TreeNode{
				Type:   domain.IDTypeFile,
				ID:     "", // Files don't have Johnny Decimal IDs
				Name:   entry.Name(),
				Path:   filepath.Join(node.Path, entry.Name()),
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

// findPathInDir looks for a directory in parentPath that starts with "id "
func findPathInDir(parentPath, id, entityType string) (string, error) {
	entries, err := os.ReadDir(parentPath)
	if err != nil {
		return "", err
	}

	prefix := id + " "
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			return filepath.Join(parentPath, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("%s not found: %s", entityType, id)
}

func (r *Repository) findScopePath(scopeID string) (string, error) {
	return findPathInDir(r.vaultPath, scopeID, "scope")
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
	return findPathInDir(scopePath, areaID, "area")
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
	return findPathInDir(areaPath, categoryID, "category")
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
	return findPathInDir(categoryPath, itemID, "item")
}
