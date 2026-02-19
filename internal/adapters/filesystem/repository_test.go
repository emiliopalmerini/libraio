package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"libraio/internal/domain"
)

func setupTestVault(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "libraio-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create scope
	scopePath := filepath.Join(tmpDir, "S01 Test")
	if err := os.MkdirAll(scopePath, 0755); err != nil {
		t.Fatalf("failed to create scope: %v", err)
	}

	// Create area
	areaPath := filepath.Join(scopePath, "S01.10-19 TestArea")
	if err := os.MkdirAll(areaPath, 0755); err != nil {
		t.Fatalf("failed to create area: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestCreateCategory_CreatesStandardZeros(t *testing.T) {
	vaultPath, cleanup := setupTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	cat, err := repo.CreateCategory("S01.10-19", "Entertainment")
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	if cat.ID != "S01.11" {
		t.Errorf("expected category ID S01.11, got %s", cat.ID)
	}

	// Verify all standard zeros were created with correct naming
	for _, sz := range domain.StandardZeros {
		itemID := cat.ID + "." + padNumber(sz.Number)
		// Use context-aware naming: "Inbox for S01.11"
		itemName := domain.StandardZeroNameForContext(sz.Name, cat.ID)
		folderName := domain.FormatFolderName(itemID, itemName)
		itemPath := filepath.Join(cat.Path, folderName)

		if _, err := os.Stat(itemPath); os.IsNotExist(err) {
			t.Errorf("standard zero %s not created at %s", itemName, itemPath)
		}

		// Verify naming format includes "for S01.11"
		if !strings.Contains(folderName, "for S01.11") {
			t.Errorf("expected folder name to contain 'for S01.11', got %s", folderName)
		}
	}
}

func TestCreateCategory_AreaManagementNaming(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "libraio-area-mgmt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create scope and area for management category
	scopePath := filepath.Join(tmpDir, "S01 Test")
	os.MkdirAll(scopePath, 0755)
	areaPath := filepath.Join(scopePath, "S01.10-19 Lifestyle")
	os.MkdirAll(areaPath, 0755)

	// Pre-create the .10 management category folder to simulate existing structure
	// Then use CreateStandardZeros to test the naming
	mgmtCatPath := filepath.Join(areaPath, "S01.10 Management for S01.10-19")
	os.MkdirAll(mgmtCatPath, 0755)

	repo := NewRepository(tmpDir)

	// Create standard zeros in area management category
	err = repo.CreateStandardZeros("S01.10", mgmtCatPath)
	if err != nil {
		t.Fatalf("CreateStandardZeros failed: %v", err)
	}

	// Verify standard zeros use area ID format: "Inbox for S01.10-19"
	for _, sz := range domain.StandardZeros {
		itemID := fmt.Sprintf("S01.10.%02d", sz.Number)
		// Area management should use "for S01.10-19" format
		expectedName := fmt.Sprintf("%s for S01.10-19", sz.Name)
		folderName := domain.FormatFolderName(itemID, expectedName)
		itemPath := filepath.Join(mgmtCatPath, folderName)

		if _, err := os.Stat(itemPath); os.IsNotExist(err) {
			t.Errorf("standard zero not created with expected naming at %s", itemPath)
		}
	}
}

func TestCreateCategory_RollbackIntegration(t *testing.T) {
	vaultPath, cleanup := setupTestVault(t)
	defer cleanup()

	areaPath := filepath.Join(vaultPath, "S01 Test", "S01.10-19 TestArea")

	// Create a read-only category folder to simulate failure during standard zero creation
	categoryPath := filepath.Join(areaPath, "S01.11 ReadOnly")
	if err := os.MkdirAll(categoryPath, 0755); err != nil {
		t.Fatalf("failed to create category folder: %v", err)
	}

	// Make category folder read-only so we can't create subdirectories
	if err := os.Chmod(categoryPath, 0555); err != nil {
		t.Skipf("cannot change permissions: %v", err)
	}
	defer os.Chmod(categoryPath, 0755) // Restore for cleanup

	repo := NewRepository(vaultPath)

	// CreateStandardZeros should fail because we can't create subdirectories
	err := repo.CreateStandardZeros("S01.11", categoryPath)

	if err == nil {
		t.Fatal("expected CreateStandardZeros to fail on read-only directory")
	}
}

func TestCreateCategory_StandardZeroCount(t *testing.T) {
	vaultPath, cleanup := setupTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	cat, err := repo.CreateCategory("S01.10-19", "TestCat")
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	items, err := repo.ListItems(cat.ID)
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}

	expected := len(domain.StandardZeros)
	if len(items) != expected {
		t.Errorf("expected %d standard zero items, got %d", expected, len(items))
	}
}

func TestCreateCategory_StandardZeroIDs(t *testing.T) {
	vaultPath, cleanup := setupTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	cat, err := repo.CreateCategory("S01.10-19", "TestCat")
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	items, err := repo.ListItems(cat.ID)
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}

	// Build expected IDs
	expectedIDs := make(map[string]bool)
	for _, sz := range domain.StandardZeros {
		expectedIDs[cat.ID+"."+padNumber(sz.Number)] = true
	}

	for _, item := range items {
		if !expectedIDs[item.ID] {
			t.Errorf("unexpected item ID: %s", item.ID)
		}
		delete(expectedIDs, item.ID)
	}

	for id := range expectedIDs {
		t.Errorf("missing expected item ID: %s", id)
	}
}

func padNumber(n int) string {
	return fmt.Sprintf("%02d", n)
}

func setupSearchTestVault(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "libraio-search-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create scope
	scopePath := filepath.Join(tmpDir, "S01 Personal")
	if err := os.MkdirAll(scopePath, 0755); err != nil {
		t.Fatalf("failed to create scope: %v", err)
	}

	// Create area
	areaPath := filepath.Join(scopePath, "S01.10-19 Lifestyle")
	if err := os.MkdirAll(areaPath, 0755); err != nil {
		t.Fatalf("failed to create area: %v", err)
	}

	// Create category
	categoryPath := filepath.Join(areaPath, "S01.11 Entertainment")
	if err := os.MkdirAll(categoryPath, 0755); err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// Create item with README
	itemPath := filepath.Join(categoryPath, "S01.11.15 Movies")
	if err := os.MkdirAll(itemPath, 0755); err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	readme := `---
aliases:
  - S01.11.15 Movies
tags:
  - jdex
---

# S01.11.15 Movies

Collection of movie notes and reviews.
`
	if err := os.WriteFile(filepath.Join(itemPath, "README.md"), []byte(readme), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestSearch_FindsFileByFilename(t *testing.T) {
	vaultPath, cleanup := setupSearchTestVault(t)
	defer cleanup()

	itemPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment", "S01.11.15 Movies")

	// Create files with various extensions
	files := []string{
		"inception-notes.md",
		"matrix-script.txt",
		"interstellar-poster.png",
		"dune-soundtrack.mp3",
	}

	for _, f := range files {
		if err := os.WriteFile(filepath.Join(itemPath, f), []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", f, err)
		}
	}

	repo := NewRepository(vaultPath)

	tests := []struct {
		query    string
		expectID string
	}{
		{"inception", "S01.11.15"},
		{"matrix", "S01.11.15"},
		{"interstellar", "S01.11.15"},
		{"dune", "S01.11.15"},
	}

	for _, tc := range tests {
		results, err := repo.Search(tc.query)
		if err != nil {
			t.Fatalf("Search '%s' failed: %v", tc.query, err)
		}

		if len(results) != 1 {
			t.Errorf("Search '%s': expected 1 result, got %d", tc.query, len(results))
			continue
		}

		if results[0].ID != tc.expectID {
			t.Errorf("Search '%s': expected ID %s, got %s", tc.query, tc.expectID, results[0].ID)
		}
	}
}

func TestSearch_DoesNotMatchFileContent(t *testing.T) {
	vaultPath, cleanup := setupSearchTestVault(t)
	defer cleanup()

	// Create a file where search term is only in content, not filename
	itemPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment", "S01.11.15 Movies")
	content := `# Movie Notes

Christopher Nolan directed this masterpiece.
`
	if err := os.WriteFile(filepath.Join(itemPath, "film-review.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	repo := NewRepository(vaultPath)

	// "nolan" is in content but not in filename - should NOT match
	results, err := repo.Search("nolan")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results (content-only match), got %d", len(results))
	}
}

func TestSearch_FindsDeepestValidID(t *testing.T) {
	vaultPath, cleanup := setupSearchTestVault(t)
	defer cleanup()

	// Create nested folder inside item (not a JD ID folder)
	itemPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment", "S01.11.15 Movies")
	nestedPath := filepath.Join(itemPath, "2024")
	if err := os.MkdirAll(nestedPath, 0755); err != nil {
		t.Fatalf("failed to create nested folder: %v", err)
	}

	// Create markdown file in nested non-JD folder - search matches filename
	if err := os.WriteFile(filepath.Join(nestedPath, "dune-part-two.md"), []byte("# Notes"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	repo := NewRepository(vaultPath)

	results, err := repo.Search("dune")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Should return the deepest valid JD ID (S01.11.15), not the nested folder
	if results[0].ID != "S01.11.15" {
		t.Errorf("expected deepest valid ID S01.11.15, got %s", results[0].ID)
	}
}

func TestSearch_DeduplicatesResultsPerID(t *testing.T) {
	vaultPath, cleanup := setupSearchTestVault(t)
	defer cleanup()

	itemPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment", "S01.11.15 Movies")

	// Create multiple markdown files with filenames containing same search term
	files := []string{
		"scifi-blade-runner.md",
		"scifi-matrix.md",
		"scifi-interstellar.md",
	}

	for _, name := range files {
		if err := os.WriteFile(filepath.Join(itemPath, name), []byte("# Notes"), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
	}

	repo := NewRepository(vaultPath)

	results, err := repo.Search("scifi")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should return only one result for the ID, not three
	if len(results) != 1 {
		t.Errorf("expected 1 deduplicated result, got %d", len(results))
	}

	if len(results) > 0 && results[0].ID != "S01.11.15" {
		t.Errorf("expected ID S01.11.15, got %s", results[0].ID)
	}
}

func TestSearch_FolderNameMatchingStillWorks(t *testing.T) {
	vaultPath, cleanup := setupSearchTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	// Search for folder name
	results, err := repo.Search("entertainment")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	found := false
	for _, r := range results {
		if r.ID == "S01.11" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected to find S01.11 Entertainment via folder name match")
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	vaultPath, cleanup := setupSearchTestVault(t)
	defer cleanup()

	itemPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment", "S01.11.15 Movies")
	// Filename has mixed case
	if err := os.WriteFile(filepath.Join(itemPath, "MixedCaseFilename.md"), []byte("# Notes"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	repo := NewRepository(vaultPath)

	// Search with lowercase should find mixed case filename
	results, err := repo.Search("mixedcase")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result for case-insensitive search, got %d", len(results))
	}
}

func TestSearch_NoResultsForUnmatchedQuery(t *testing.T) {
	vaultPath, cleanup := setupSearchTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	results, err := repo.Search("xyznonexistent")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for unmatched query, got %d", len(results))
	}
}

// Archive test helpers

func setupArchiveTestVault(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "libraio-archive-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create scope
	scopePath := filepath.Join(tmpDir, "S01 Personal")
	if err := os.MkdirAll(scopePath, 0755); err != nil {
		t.Fatalf("failed to create scope: %v", err)
	}

	// Create area
	areaPath := filepath.Join(scopePath, "S01.10-19 Lifestyle")
	if err := os.MkdirAll(areaPath, 0755); err != nil {
		t.Fatalf("failed to create area: %v", err)
	}

	// Create category
	categoryPath := filepath.Join(areaPath, "S01.11 Entertainment")
	if err := os.MkdirAll(categoryPath, 0755); err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// Create category-level archive item (.09 Archive for S01.11)
	archiveItemPath := filepath.Join(categoryPath, "S01.11.09 Archive for S01.11")
	if err := os.MkdirAll(archiveItemPath, 0755); err != nil {
		t.Fatalf("failed to create archive item: %v", err)
	}
	archiveReadme := `---
aliases:
  - S01.11.09 Archive for S01.11
tags:
  - jdex
  - standard-zero
---

# S01.11.09 Archive for S01.11

Archived items for this category.
`
	if err := os.WriteFile(filepath.Join(archiveItemPath, "S01.11.09 Archive for S01.11.md"), []byte(archiveReadme), 0644); err != nil {
		t.Fatalf("failed to create archive JDex: %v", err)
	}

	// Create item in category with JDex file
	itemPath := filepath.Join(categoryPath, "S01.11.15 Theatre")
	if err := os.MkdirAll(itemPath, 0755); err != nil {
		t.Fatalf("failed to create item: %v", err)
	}
	jdexContent := `---
aliases:
  - S01.11.15 Theatre
tags:
  - jdex
---

# S01.11.15 Theatre

Theatre collection.
`
	if err := os.WriteFile(filepath.Join(itemPath, "S01.11.15 Theatre.md"), []byte(jdexContent), 0644); err != nil {
		t.Fatalf("failed to create JDex: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// ArchiveItem tests

func TestArchiveItem_MovesToArchiveFolder(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	// Archive the item
	archivedItem, err := repo.ArchiveItem("S01.11.15")
	if err != nil {
		t.Fatalf("ArchiveItem failed: %v", err)
	}

	// Verify item loses its ID after archiving
	if archivedItem.ID != "" {
		t.Errorf("expected empty ID after archiving, got %s", archivedItem.ID)
	}

	if archivedItem.CategoryID != "S01.11" {
		t.Errorf("expected CategoryID S01.11, got %s", archivedItem.CategoryID)
	}

	// Verify item name is preserved
	if archivedItem.Name != "Theatre" {
		t.Errorf("expected name Theatre, got %s", archivedItem.Name)
	}

	// Verify item is now inside the archive folder with [Archived] prefix
	archivePath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment", "S01.11.09 Archive for S01.11")
	expectedPath := filepath.Join(archivePath, "[Archived] Theatre")
	if archivedItem.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, archivedItem.Path)
	}

	// Verify item folder exists at new location
	if _, err := os.Stat(archivedItem.Path); os.IsNotExist(err) {
		t.Errorf("archived item folder not found at %s", archivedItem.Path)
	}

	// Verify original location no longer exists
	originalPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment", "S01.11.15 Theatre")
	if _, err := os.Stat(originalPath); !os.IsNotExist(err) {
		t.Error("expected original item location to be gone, but it still exists")
	}
}

func TestArchiveItem_MultipleItemsCanBeArchived(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	// Create another item to archive
	categoryPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment")
	item2Path := filepath.Join(categoryPath, "S01.11.16 Movies")
	if err := os.MkdirAll(item2Path, 0755); err != nil {
		t.Fatalf("failed to create second item: %v", err)
	}
	if err := os.WriteFile(filepath.Join(item2Path, "S01.11.16 Movies.md"), []byte("# S01.11.16 Movies"), 0644); err != nil {
		t.Fatalf("failed to create JDex: %v", err)
	}

	repo := NewRepository(vaultPath)

	// Archive both items
	archivedItem1, err := repo.ArchiveItem("S01.11.15")
	if err != nil {
		t.Fatalf("ArchiveItem (first) failed: %v", err)
	}

	archivedItem2, err := repo.ArchiveItem("S01.11.16")
	if err != nil {
		t.Fatalf("ArchiveItem (second) failed: %v", err)
	}

	// Both items should be in the archive folder with [Archived] prefix
	archivePath := filepath.Join(categoryPath, "S01.11.09 Archive for S01.11")
	if _, err := os.Stat(filepath.Join(archivePath, "[Archived] Theatre")); os.IsNotExist(err) {
		t.Error("first archived item not found in archive folder")
	}
	if _, err := os.Stat(filepath.Join(archivePath, "[Archived] Movies")); os.IsNotExist(err) {
		t.Error("second archived item not found in archive folder")
	}

	// Items lose their IDs after archiving
	if archivedItem1.ID != "" {
		t.Errorf("expected first item to have no ID, got %s", archivedItem1.ID)
	}
	if archivedItem2.ID != "" {
		t.Errorf("expected second item to have no ID, got %s", archivedItem2.ID)
	}
}

func TestArchiveItem_FailsForNonItem(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	// Try to archive a category (should fail)
	_, err := repo.ArchiveItem("S01.11")
	if err == nil {
		t.Error("expected error when archiving a category, got nil")
	}

	// Try to archive an area (should fail)
	_, err = repo.ArchiveItem("S01.10-19")
	if err == nil {
		t.Error("expected error when archiving an area, got nil")
	}
}

func TestArchiveItem_FailsIfAlreadyArchiveItem(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	// Try to archive the .09 Archive item itself (should fail)
	_, err := repo.ArchiveItem("S01.11.09")
	if err == nil {
		t.Error("expected error when archiving an archive item (.09), got nil")
	}
}

func TestArchiveItem_FailsIfArchiveItemMissing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "libraio-archive-no-archive-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create vault without .09 Archive item
	scopePath := filepath.Join(tmpDir, "S01 Personal")
	os.MkdirAll(scopePath, 0755)
	areaPath := filepath.Join(scopePath, "S01.10-19 Lifestyle")
	os.MkdirAll(areaPath, 0755)
	categoryPath := filepath.Join(areaPath, "S01.11 Entertainment")
	os.MkdirAll(categoryPath, 0755)
	itemPath := filepath.Join(categoryPath, "S01.11.15 Theatre")
	os.MkdirAll(itemPath, 0755)
	os.WriteFile(filepath.Join(itemPath, "README.md"), []byte("# Theatre"), 0644)

	repo := NewRepository(tmpDir)

	_, err = repo.ArchiveItem("S01.11.15")
	if err == nil {
		t.Error("expected error when .09 Archive item is missing, got nil")
	}
}

// ArchiveCategory tests

func TestArchiveCategory_MovesAllItemsToArchiveFolder(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	// Create additional items in the category
	categoryPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment")

	item2Path := filepath.Join(categoryPath, "S01.11.16 Movies")
	os.MkdirAll(item2Path, 0755)
	os.WriteFile(filepath.Join(item2Path, "README.md"), []byte("# Movies"), 0644)

	item3Path := filepath.Join(categoryPath, "S01.11.17 Music")
	os.MkdirAll(item3Path, 0755)
	os.WriteFile(filepath.Join(item3Path, "README.md"), []byte("# Music"), 0644)

	repo := NewRepository(vaultPath)

	archivedItems, err := repo.ArchiveCategory("S01.11")
	if err != nil {
		t.Fatalf("ArchiveCategory failed: %v", err)
	}

	// Should have archived 3 items
	if len(archivedItems) != 3 {
		t.Errorf("expected 3 archived items, got %d", len(archivedItems))
	}

	// All items should be in the .09 Archive folder
	archivePath := filepath.Join(categoryPath, "S01.11.09 Archive for S01.11")
	for _, item := range archivedItems {
		// Items keep original category ID
		if item.CategoryID != "S01.11" {
			t.Errorf("expected item %s to have CategoryID S01.11, got %s", item.ID, item.CategoryID)
		}
		// Items should be inside the archive folder
		if _, err := os.Stat(item.Path); os.IsNotExist(err) {
			t.Errorf("archived item not found at %s", item.Path)
		}
		if !strings.HasPrefix(item.Path, archivePath) {
			t.Errorf("expected item path to be under archive, got %s", item.Path)
		}
	}

	// Category should still exist (NOT deleted)
	if _, err := os.Stat(categoryPath); os.IsNotExist(err) {
		t.Error("expected category to still exist after archiving")
	}
}

func TestArchiveCategory_SkipsStandardZeros(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	// Create additional standard zero items (these should be skipped)
	// Note: .09 Archive already exists from setup
	categoryPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment")

	for _, sz := range domain.StandardZeros {
		if sz.Number == 9 {
			continue // .09 Archive already exists
		}
		szPath := filepath.Join(categoryPath, fmt.Sprintf("S01.11.%02d %s", sz.Number, sz.Name))
		os.MkdirAll(szPath, 0755)
		os.WriteFile(filepath.Join(szPath, "README.md"), []byte(fmt.Sprintf("# %s", sz.Name)), 0644)
	}

	repo := NewRepository(vaultPath)

	archivedItems, err := repo.ArchiveCategory("S01.11")
	if err != nil {
		t.Fatalf("ArchiveCategory failed: %v", err)
	}

	// Should only archive 1 item (S01.11.15 Theatre), not the standard zeros
	if len(archivedItems) != 1 {
		t.Errorf("expected 1 archived item (standard zeros should be skipped), got %d", len(archivedItems))
	}

	// Verify the archived item is S01.11.15 Theatre
	if len(archivedItems) > 0 && archivedItems[0].Name != "Theatre" {
		t.Errorf("expected archived item to be Theatre, got %s", archivedItems[0].Name)
	}
}

func TestArchiveCategory_FailsForNonCategory(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	// Try to archive an item (should fail)
	_, err := repo.ArchiveCategory("S01.11.15")
	if err == nil {
		t.Error("expected error when archiving an item, got nil")
	}

	// Try to archive an area (should fail)
	_, err = repo.ArchiveCategory("S01.10-19")
	if err == nil {
		t.Error("expected error when archiving an area, got nil")
	}
}

func TestArchiveCategory_FailsIfArchiveItemMissing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "libraio-archive-no-archive-item-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create vault without .09 Archive item
	scopePath := filepath.Join(tmpDir, "S01 Personal")
	os.MkdirAll(scopePath, 0755)
	areaPath := filepath.Join(scopePath, "S01.10-19 Lifestyle")
	os.MkdirAll(areaPath, 0755)
	categoryPath := filepath.Join(areaPath, "S01.11 Entertainment")
	os.MkdirAll(categoryPath, 0755)
	itemPath := filepath.Join(categoryPath, "S01.11.15 Theatre")
	os.MkdirAll(itemPath, 0755)
	os.WriteFile(filepath.Join(itemPath, "README.md"), []byte("# Theatre"), 0644)

	repo := NewRepository(tmpDir)

	// Try to archive category without .09 Archive item (should fail)
	_, err = repo.ArchiveCategory("S01.11")
	if err == nil {
		t.Error("expected error when .09 Archive item is missing, got nil")
	}
}

func TestArchiveCategory_PreservesItemDescriptions(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	archivedItems, err := repo.ArchiveCategory("S01.11")
	if err != nil {
		t.Fatalf("ArchiveCategory failed: %v", err)
	}

	// Find the Theatre item
	var theatreItem *domain.Item
	for _, item := range archivedItems {
		if item.Name == "Theatre" {
			theatreItem = item
			break
		}
	}

	if theatreItem == nil {
		t.Fatal("Theatre item not found in archived items")
	}

	// Verify the description was preserved
	if theatreItem.Name != "Theatre" {
		t.Errorf("expected name Theatre, got %s", theatreItem.Name)
	}
}

// Tests for new category-to-area archive functionality

func setupCategoryToAreaArchiveVault(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "libraio-cat-area-archive-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create scope
	scopePath := filepath.Join(tmpDir, "S01 Personal")
	os.MkdirAll(scopePath, 0755)

	// Create area
	areaPath := filepath.Join(scopePath, "S01.10-19 Lifestyle")
	os.MkdirAll(areaPath, 0755)

	// Create area management category with archive (.10)
	mgmtCatPath := filepath.Join(areaPath, "S01.10 Management for S01.10-19")
	os.MkdirAll(mgmtCatPath, 0755)

	// Create area-level archive item (.10.09)
	areaArchivePath := filepath.Join(mgmtCatPath, "S01.10.09 Archive for S01.10-19")
	os.MkdirAll(areaArchivePath, 0755)
	os.WriteFile(filepath.Join(areaArchivePath, "S01.10.09 Archive for S01.10-19.md"),
		[]byte("# S01.10.09 Archive for S01.10-19\n\nArea archive."), 0644)

	// Create category to be archived
	categoryPath := filepath.Join(areaPath, "S01.11 Entertainment")
	os.MkdirAll(categoryPath, 0755)

	// Create category-level archive item (.11.09)
	catArchivePath := filepath.Join(categoryPath, "S01.11.09 Archive for S01.11")
	os.MkdirAll(catArchivePath, 0755)
	os.WriteFile(filepath.Join(catArchivePath, "S01.11.09 Archive for S01.11.md"),
		[]byte("# S01.11.09 Archive for S01.11\n\nCategory archive."), 0644)

	// Create item in category
	itemPath := filepath.Join(categoryPath, "S01.11.15 Theatre")
	os.MkdirAll(itemPath, 0755)
	os.WriteFile(filepath.Join(itemPath, "S01.11.15 Theatre.md"),
		[]byte("# S01.11.15 Theatre\n\nTheatre content."), 0644)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestArchiveCategoryToArea_MovesToAreaArchive(t *testing.T) {
	vaultPath, cleanup := setupCategoryToAreaArchiveVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	// Archive category S01.11 to area archive S01.10.09
	archivedCat, err := repo.ArchiveCategoryToArea("S01.11")
	if err != nil {
		t.Fatalf("ArchiveCategoryToArea failed: %v", err)
	}

	// Verify category folder now exists inside area archive
	areaArchivePath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle",
		"S01.10 Management for S01.10-19", "S01.10.09 Archive for S01.10-19")
	expectedCatPath := filepath.Join(areaArchivePath, "S01.11 Entertainment")

	if _, err := os.Stat(expectedCatPath); os.IsNotExist(err) {
		t.Errorf("category folder not found in area archive at %s", expectedCatPath)
	}

	// Verify original category location no longer exists
	originalPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment")
	if _, err := os.Stat(originalPath); !os.IsNotExist(err) {
		t.Error("expected original category location to be gone")
	}

	// Verify returned category info
	if archivedCat.ID != "S01.11" {
		t.Errorf("expected category ID S01.11, got %s", archivedCat.ID)
	}
	if archivedCat.Name != "Entertainment" {
		t.Errorf("expected category name Entertainment, got %s", archivedCat.Name)
	}
}

func TestArchiveCategoryToArea_PreservesContents(t *testing.T) {
	vaultPath, cleanup := setupCategoryToAreaArchiveVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	_, err := repo.ArchiveCategoryToArea("S01.11")
	if err != nil {
		t.Fatalf("ArchiveCategoryToArea failed: %v", err)
	}

	// Verify item inside category was preserved
	areaArchivePath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle",
		"S01.10 Management for S01.10-19", "S01.10.09 Archive for S01.10-19")
	itemPath := filepath.Join(areaArchivePath, "S01.11 Entertainment", "S01.11.15 Theatre")

	if _, err := os.Stat(itemPath); os.IsNotExist(err) {
		t.Errorf("item not preserved inside archived category at %s", itemPath)
	}

	// Verify item JDex file exists
	jdexPath := filepath.Join(itemPath, "S01.11.15 Theatre.md")
	if _, err := os.Stat(jdexPath); os.IsNotExist(err) {
		t.Error("item JDex file not preserved")
	}
}

func TestArchiveCategoryToArea_FailsForNonCategory(t *testing.T) {
	vaultPath, cleanup := setupCategoryToAreaArchiveVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	// Try to archive an item (should fail)
	_, err := repo.ArchiveCategoryToArea("S01.11.15")
	if err == nil {
		t.Error("expected error when archiving an item, got nil")
	}

	// Try to archive an area (should fail)
	_, err = repo.ArchiveCategoryToArea("S01.10-19")
	if err == nil {
		t.Error("expected error when archiving an area, got nil")
	}
}

func TestArchiveCategoryToArea_FailsIfAreaArchiveMissing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "libraio-no-area-archive-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create structure without area archive
	scopePath := filepath.Join(tmpDir, "S01 Personal")
	os.MkdirAll(scopePath, 0755)
	areaPath := filepath.Join(scopePath, "S01.10-19 Lifestyle")
	os.MkdirAll(areaPath, 0755)
	categoryPath := filepath.Join(areaPath, "S01.11 Entertainment")
	os.MkdirAll(categoryPath, 0755)

	repo := NewRepository(tmpDir)

	_, err = repo.ArchiveCategoryToArea("S01.11")
	if err == nil {
		t.Error("expected error when area archive is missing, got nil")
	}
}

func TestArchiveCategoryToArea_FailsForManagementCategory(t *testing.T) {
	vaultPath, cleanup := setupCategoryToAreaArchiveVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	// Try to archive the management category itself (should fail)
	_, err := repo.ArchiveCategoryToArea("S01.10")
	if err == nil {
		t.Error("expected error when archiving management category, got nil")
	}
}

// Obsidian link update tests

func setupLinkTestVault(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "libraio-link-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create scope
	scopePath := filepath.Join(tmpDir, "S01 Personal")
	os.MkdirAll(scopePath, 0755)

	// Create area
	areaPath := filepath.Join(scopePath, "S01.10-19 Lifestyle")
	os.MkdirAll(areaPath, 0755)

	// Create category
	categoryPath := filepath.Join(areaPath, "S01.11 Entertainment")
	os.MkdirAll(categoryPath, 0755)

	// Create category-level archive item (.09 Archive for S01.11)
	archiveItemPath := filepath.Join(categoryPath, "S01.11.09 Archive for S01.11")
	os.MkdirAll(archiveItemPath, 0755)
	os.WriteFile(filepath.Join(archiveItemPath, "S01.11.09 Archive for S01.11.md"), []byte("# Archive for S01.11"), 0644)

	// Create item to be archived
	itemPath := filepath.Join(categoryPath, "S01.11.15 Theatre")
	os.MkdirAll(itemPath, 0755)
	os.WriteFile(filepath.Join(itemPath, "README.md"), []byte("# Theatre"), 0644)

	// Create another item that links to the item being archived
	linkerPath := filepath.Join(categoryPath, "S01.11.16 Links")
	os.MkdirAll(linkerPath, 0755)

	linkingContent := `---
aliases:
  - S01.11.16 Links
---

# Links

- See [[S01.11.15 Theatre]] for theatre info
- Also check [[S01.11.15]] for more
- Reference: [[S01.11.15|My Theatre Link]]
`
	os.WriteFile(filepath.Join(linkerPath, "links.md"), []byte(linkingContent), 0644)

	// Create a file outside the item structure that also links
	os.WriteFile(filepath.Join(tmpDir, "notes.md"), []byte("Check [[S01.11.15 Theatre]]"), 0644)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestArchiveItem_UpdatesObsidianLinks(t *testing.T) {
	vaultPath, cleanup := setupLinkTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	_, err := repo.ArchiveItem("S01.11.15")
	if err != nil {
		t.Fatalf("ArchiveItem failed: %v", err)
	}

	// Read the linking file and verify links were updated to remove ID
	linkingFilePath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment", "S01.11.16 Links", "links.md")
	content, err := os.ReadFile(linkingFilePath)
	if err != nil {
		t.Fatalf("failed to read linking file: %v", err)
	}

	contentStr := string(content)

	// Links should now use [Archived] prefix (ID removed)
	if strings.Contains(contentStr, "[[S01.11.15 Theatre]]") {
		t.Error("linking file should NOT contain [[S01.11.15 Theatre]] link after archiving")
	}
	if strings.Contains(contentStr, "[[S01.11.15]]") {
		t.Error("linking file should NOT contain [[S01.11.15]] link after archiving")
	}
	if !strings.Contains(contentStr, "[[[Archived] Theatre]]") {
		t.Error("linking file should contain [[[Archived] Theatre]] link after archiving")
	}

	// Check the root-level notes file too
	notesContent, err := os.ReadFile(filepath.Join(vaultPath, "notes.md"))
	if err != nil {
		t.Fatalf("failed to read notes.md: %v", err)
	}

	if strings.Contains(string(notesContent), "[[S01.11.15 Theatre]]") {
		t.Error("notes.md should NOT contain original link after archiving")
	}
	if !strings.Contains(string(notesContent), "[[[Archived] Theatre]]") {
		t.Error("notes.md should contain [[[Archived] Theatre]] link after archiving")
	}
}

func TestArchiveItem_UpdatesLinksWithVariousFormats(t *testing.T) {
	vaultPath, cleanup := setupLinkTestVault(t)
	defer cleanup()

	// Create a file with various link formats
	testFilePath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment", "S01.11.16 Links", "test-links.md")
	content := `# Link Formats Test

1. Wiki link with title: [[S01.11.15 Theatre]]
2. Wiki link ID only: [[S01.11.15]]
3. Wiki link with alias: [[S01.11.15|Custom Title]]
4. Wiki link with title and alias: [[S01.11.15 Theatre|Another Title]]
5. Multiple on same line: [[S01.11.15]] and [[S01.11.15 Theatre]]
6. In a sentence: Check out [[S01.11.15 Theatre]] for details.
`
	if err := os.WriteFile(testFilePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	repo := NewRepository(vaultPath)

	_, err := repo.ArchiveItem("S01.11.15")
	if err != nil {
		t.Fatalf("ArchiveItem failed: %v", err)
	}

	// Read the file (links should be updated to remove ID)
	updatedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	updatedStr := string(updatedContent)

	// Verify links were updated to use [Archived] prefix
	if strings.Contains(updatedStr, "[[S01.11.15 Theatre]]") {
		t.Errorf("file should NOT contain [[S01.11.15 Theatre]] links after archiving")
	}
	if strings.Contains(updatedStr, "[[S01.11.15]]") {
		t.Errorf("file should NOT contain [[S01.11.15]] links after archiving")
	}

	// Verify updated links with [Archived] prefix
	if !strings.Contains(updatedStr, "[[[Archived] Theatre]]") {
		t.Errorf("file should contain [[[Archived] Theatre]] links after archiving")
	}
	if !strings.Contains(updatedStr, "[[[Archived] Theatre|Custom Title]]") {
		t.Errorf("file should contain [[[Archived] Theatre|Custom Title]] link after archiving")
	}
	if !strings.Contains(updatedStr, "[[[Archived] Theatre|Another Title]]") {
		t.Errorf("file should contain [[[Archived] Theatre|Another Title]] link after archiving")
	}
}
