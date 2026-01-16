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
