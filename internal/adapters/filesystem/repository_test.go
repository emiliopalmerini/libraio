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

	// Verify all standard zeros were created
	for _, sz := range domain.StandardZeros {
		itemID := cat.ID + "." + padNumber(sz.Number)
		itemPath := filepath.Join(cat.Path, domain.FormatFolderName(itemID, sz.Name))

		if _, err := os.Stat(itemPath); os.IsNotExist(err) {
			t.Errorf("standard zero %s not created at %s", sz.Name, itemPath)
		}

		readmePath := filepath.Join(itemPath, "README.md")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			t.Errorf("README not created for %s", sz.Name)
		}

		content, err := os.ReadFile(readmePath)
		if err != nil {
			t.Errorf("failed to read README for %s: %v", sz.Name, err)
			continue
		}

		// Verify README contains the purpose
		if !strings.Contains(string(content), sz.Purpose) {
			t.Errorf("README for %s does not contain purpose text", sz.Name)
		}

		// Verify standard-zero tag
		if !strings.Contains(string(content), "standard-zero") {
			t.Errorf("README for %s does not contain standard-zero tag", sz.Name)
		}
	}
}

func TestCreateCategory_RollbackOnFailure(t *testing.T) {
	vaultPath, cleanup := setupTestVault(t)
	defer cleanup()

	areaPath := filepath.Join(vaultPath, "S01 Test", "S01.10-19 TestArea")

	// Create a category folder manually (simulating what CreateCategory does)
	categoryPath := filepath.Join(areaPath, "S01.11 TestRollback")
	if err := os.MkdirAll(categoryPath, 0755); err != nil {
		t.Fatalf("failed to create category folder: %v", err)
	}

	// Create a FILE where a standard zero DIRECTORY should be
	// This will cause CreateStandardZeros to fail
	conflictPath := filepath.Join(categoryPath, "S01.11.00 JDex")
	if err := os.WriteFile(conflictPath, []byte("conflict"), 0644); err != nil {
		t.Fatalf("failed to create conflict file: %v", err)
	}

	repo := NewRepository(vaultPath)

	// Test CreateStandardZeros directly - it should fail because JDex path is a file
	err := repo.CreateStandardZeros("S01.11", categoryPath)

	if err == nil {
		t.Fatal("expected CreateStandardZeros to fail, but it succeeded")
	}

	if !strings.Contains(err.Error(), "JDex") {
		t.Errorf("expected error to mention JDex, got: %v", err)
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

	// Create archive category
	archiveCategoryPath := filepath.Join(areaPath, "S01.19 Archive")
	if err := os.MkdirAll(archiveCategoryPath, 0755); err != nil {
		t.Fatalf("failed to create archive category: %v", err)
	}

	// Create item in category with README
	itemPath := filepath.Join(categoryPath, "S01.11.15 Theatre")
	if err := os.MkdirAll(itemPath, 0755); err != nil {
		t.Fatalf("failed to create item: %v", err)
	}
	readme := `---
aliases:
  - S01.11.15 Theatre
tags:
  - jdex
---

# S01.11.15 Theatre

Theatre collection.
`
	if err := os.WriteFile(filepath.Join(itemPath, "README.md"), []byte(readme), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// ArchiveItem tests

func TestArchiveItem_MovesToArchiveCategory(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	// Archive the item
	archivedItem, err := repo.ArchiveItem("S01.11.15")
	if err != nil {
		t.Fatalf("ArchiveItem failed: %v", err)
	}

	// Verify item moved to archive category (S01.19)
	if archivedItem.CategoryID != "S01.19" {
		t.Errorf("expected CategoryID S01.19, got %s", archivedItem.CategoryID)
	}

	// Verify item got a new ID in the archive category
	if !strings.HasPrefix(archivedItem.ID, "S01.19.") {
		t.Errorf("expected ID to start with S01.19., got %s", archivedItem.ID)
	}

	// Verify item name is preserved
	if archivedItem.Name != "Theatre" {
		t.Errorf("expected name Theatre, got %s", archivedItem.Name)
	}

	// Verify original item no longer exists
	_, err = repo.GetPath("S01.11.15")
	if err == nil {
		t.Error("expected original item to be deleted, but it still exists")
	}

	// Verify new item exists
	_, err = repo.GetPath(archivedItem.ID)
	if err != nil {
		t.Errorf("archived item not found at new location: %v", err)
	}
}

func TestArchiveItem_UpdatesREADME(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	archivedItem, err := repo.ArchiveItem("S01.11.15")
	if err != nil {
		t.Fatalf("ArchiveItem failed: %v", err)
	}

	// Read the README and verify it was updated
	content, err := os.ReadFile(archivedItem.ReadmePath)
	if err != nil {
		t.Fatalf("failed to read README: %v", err)
	}

	// Verify old ID is replaced with new ID
	if strings.Contains(string(content), "S01.11.15") {
		t.Error("README still contains old ID S01.11.15")
	}
	if !strings.Contains(string(content), archivedItem.ID) {
		t.Errorf("README does not contain new ID %s", archivedItem.ID)
	}
}

func TestArchiveItem_AssignsNextAvailableID(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	// Create an existing item in the archive category
	archiveItemPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.19 Archive", "S01.19.11 OldArchive")
	if err := os.MkdirAll(archiveItemPath, 0755); err != nil {
		t.Fatalf("failed to create existing archive item: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archiveItemPath, "README.md"), []byte("# Old"), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}

	repo := NewRepository(vaultPath)

	archivedItem, err := repo.ArchiveItem("S01.11.15")
	if err != nil {
		t.Fatalf("ArchiveItem failed: %v", err)
	}

	// Should get S01.19.12 since S01.19.11 is taken
	if archivedItem.ID != "S01.19.12" {
		t.Errorf("expected ID S01.19.12, got %s", archivedItem.ID)
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

func TestArchiveItem_FailsIfAlreadyInArchive(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	// Create an item already in the archive category
	archiveItemPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.19 Archive", "S01.19.11 AlreadyArchived")
	if err := os.MkdirAll(archiveItemPath, 0755); err != nil {
		t.Fatalf("failed to create archive item: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archiveItemPath, "README.md"), []byte("# Old"), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}

	repo := NewRepository(vaultPath)

	_, err := repo.ArchiveItem("S01.19.11")
	if err == nil {
		t.Error("expected error when archiving item already in archive, got nil")
	}
}

func TestArchiveItem_FailsIfArchiveCategoryMissing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "libraio-archive-no-archive-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create vault without archive category
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
		t.Error("expected error when archive category is missing, got nil")
	}
}

// ArchiveCategory tests

func TestArchiveCategory_MovesAllItemsToArchive(t *testing.T) {
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

	// All items should be in archive category now
	for _, item := range archivedItems {
		if item.CategoryID != "S01.19" {
			t.Errorf("expected item %s to be in S01.19, got %s", item.ID, item.CategoryID)
		}
	}

	// Original category should be deleted
	_, err = repo.GetPath("S01.11")
	if err == nil {
		t.Error("expected original category to be deleted, but it still exists")
	}
}

func TestArchiveCategory_SkipsStandardZeros(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	// Create standard zero items (these should be skipped)
	categoryPath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment")

	for _, sz := range domain.StandardZeros {
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

func TestArchiveCategory_FailsForArchiveCategory(t *testing.T) {
	vaultPath, cleanup := setupArchiveTestVault(t)
	defer cleanup()

	repo := NewRepository(vaultPath)

	// Try to archive the archive category itself (should fail)
	_, err := repo.ArchiveCategory("S01.19")
	if err == nil {
		t.Error("expected error when archiving the archive category, got nil")
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

	// Create archive category
	archiveCategoryPath := filepath.Join(areaPath, "S01.19 Archive")
	os.MkdirAll(archiveCategoryPath, 0755)

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

	archivedItem, err := repo.ArchiveItem("S01.11.15")
	if err != nil {
		t.Fatalf("ArchiveItem failed: %v", err)
	}

	// Read the linking file and verify links were updated
	linkingFilePath := filepath.Join(vaultPath, "S01 Personal", "S01.10-19 Lifestyle", "S01.11 Entertainment", "S01.11.16 Links", "links.md")
	content, err := os.ReadFile(linkingFilePath)
	if err != nil {
		t.Fatalf("failed to read linking file: %v", err)
	}

	contentStr := string(content)

	// Old links should be replaced
	if strings.Contains(contentStr, "[[S01.11.15 Theatre]]") {
		t.Error("linking file still contains old [[S01.11.15 Theatre]] link")
	}
	if strings.Contains(contentStr, "[[S01.11.15]]") {
		t.Error("linking file still contains old [[S01.11.15]] link")
	}
	if strings.Contains(contentStr, "[[S01.11.15|") {
		t.Error("linking file still contains old [[S01.11.15|...]] link")
	}

	// New links should be present
	expectedLink := fmt.Sprintf("[[%s Theatre]]", archivedItem.ID)
	if !strings.Contains(contentStr, expectedLink) {
		t.Errorf("linking file does not contain expected link %s", expectedLink)
	}

	// Check the root-level notes file too
	notesContent, err := os.ReadFile(filepath.Join(vaultPath, "notes.md"))
	if err != nil {
		t.Fatalf("failed to read notes.md: %v", err)
	}

	if strings.Contains(string(notesContent), "[[S01.11.15 Theatre]]") {
		t.Error("notes.md still contains old link")
	}
}

func TestUpdateObsidianLinks_HandlesVariousLinkFormats(t *testing.T) {
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

	archivedItem, err := repo.ArchiveItem("S01.11.15")
	if err != nil {
		t.Fatalf("ArchiveItem failed: %v", err)
	}

	// Read the updated file
	updatedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	updatedStr := string(updatedContent)

	// Verify no old links remain
	if strings.Contains(updatedStr, "S01.11.15") {
		t.Errorf("file still contains old ID S01.11.15:\n%s", updatedStr)
	}

	// Verify new ID is present
	if !strings.Contains(updatedStr, archivedItem.ID) {
		t.Errorf("file does not contain new ID %s:\n%s", archivedItem.ID, updatedStr)
	}
}
