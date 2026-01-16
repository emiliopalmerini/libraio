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
