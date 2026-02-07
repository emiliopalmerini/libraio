package views

import (
	"os"
	"path/filepath"
	"testing"

	"libraio/internal/domain"
)

func TestFormatTreeForSearch_FiltersManagement(t *testing.T) {
	root := &domain.TreeNode{
		Type: domain.IDTypeUnknown,
		ID:   "",
		Name: "Root",
		Children: []*domain.TreeNode{
			{
				Type: domain.IDTypeScope,
				ID:   "S01",
				Name: "Me",
				Children: []*domain.TreeNode{
					{
						Type:     domain.IDTypeArea,
						ID:       "S01.00-09",
						Name:     "Management for S01",
						Children: nil, // Should be skipped entirely
					},
					{
						Type: domain.IDTypeArea,
						ID:   "S01.10-19",
						Name: "Lifestyle",
						Children: []*domain.TreeNode{
							{
								Type:     domain.IDTypeCategory,
								ID:       "S01.10",
								Name:     "Management for S01.10-19",
								Children: nil, // Should be skipped
							},
							{
								Type: domain.IDTypeCategory,
								ID:   "S01.11",
								Name: "Entertainment",
								Children: []*domain.TreeNode{
									{
										Type: domain.IDTypeItem,
										ID:   "S01.11.01",
										Name: "Inbox for S01.11",
									},
									{
										Type: domain.IDTypeItem,
										ID:   "S01.11.09",
										Name: "Archive for S01.11",
									},
									{
										Type: domain.IDTypeItem,
										ID:   "S01.11.11",
										Name: "Theatre",
									},
									{
										Type: domain.IDTypeItem,
										ID:   "S01.11.12",
										Name: "Movies",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result := FormatTreeForSearch(root)

	// Should contain regular content
	if !contains(result, "S01 Me") {
		t.Error("expected scope S01 Me in output")
	}
	if !contains(result, "S01.10-19 Lifestyle") {
		t.Error("expected area S01.10-19 Lifestyle in output")
	}
	if !contains(result, "S01.11 Entertainment") {
		t.Error("expected category S01.11 Entertainment in output")
	}
	if !contains(result, "S01.11.11 Theatre") {
		t.Error("expected item S01.11.11 Theatre in output")
	}
	if !contains(result, "S01.11.12 Movies") {
		t.Error("expected item S01.11.12 Movies in output")
	}

	// Should NOT contain management/standard zeros
	if contains(result, "S01.00-09") {
		t.Error("management area S01.00-09 should be filtered")
	}
	if contains(result, "S01.10 Management") {
		t.Error("management category S01.10 should be filtered")
	}
	if contains(result, "S01.11.01") {
		t.Error("standard zero item S01.11.01 should be filtered")
	}
	if contains(result, "S01.11.09") {
		t.Error("standard zero item S01.11.09 should be filtered")
	}
}

func TestFormatTreeForSearch_EmptyTree(t *testing.T) {
	root := &domain.TreeNode{
		Type: domain.IDTypeUnknown,
		ID:   "",
		Name: "Root",
	}

	result := FormatTreeForSearch(root)
	if result != "" {
		t.Errorf("expected empty string for empty tree, got %q", result)
	}
}

func TestReadJDexDescription(t *testing.T) {
	t.Run("with description", func(t *testing.T) {
		dir := t.TempDir()
		folderName := "S01.11.11 Theatre"
		folderPath := filepath.Join(dir, folderName)
		if err := os.Mkdir(folderPath, 0o755); err != nil {
			t.Fatal(err)
		}

		jdexContent := `---
aliases:
  - S01.11.11 Theatre
tags:
  - jdex
---

# S01.11.11 Theatre

Contains theatre tickets and show reviews.
`
		jdexFile := filepath.Join(folderPath, folderName+".md")
		if err := os.WriteFile(jdexFile, []byte(jdexContent), 0o644); err != nil {
			t.Fatal(err)
		}

		desc := readJDexDescription(folderPath)
		if desc != "Contains theatre tickets and show reviews." {
			t.Errorf("got %q, want %q", desc, "Contains theatre tickets and show reviews.")
		}
	})

	t.Run("no jdex file", func(t *testing.T) {
		dir := t.TempDir()
		desc := readJDexDescription(dir)
		if desc != "" {
			t.Errorf("expected empty string, got %q", desc)
		}
	})

	t.Run("no description content", func(t *testing.T) {
		dir := t.TempDir()
		folderName := "S01.11.12 Movies"
		folderPath := filepath.Join(dir, folderName)
		if err := os.Mkdir(folderPath, 0o755); err != nil {
			t.Fatal(err)
		}

		jdexContent := `---
tags:
  - jdex
---

# S01.11.12 Movies
`
		jdexFile := filepath.Join(folderPath, folderName+".md")
		if err := os.WriteFile(jdexFile, []byte(jdexContent), 0o644); err != nil {
			t.Fatal(err)
		}

		desc := readJDexDescription(folderPath)
		if desc != "" {
			t.Errorf("expected empty string, got %q", desc)
		}
	})

	t.Run("truncates long descriptions", func(t *testing.T) {
		dir := t.TempDir()
		folderName := "S01.11.13 Books"
		folderPath := filepath.Join(dir, folderName)
		if err := os.Mkdir(folderPath, 0o755); err != nil {
			t.Fatal(err)
		}

		longDesc := "This is a very long description that exceeds the one hundred character limit and should be truncated to fit within the search prompt"
		jdexContent := "---\ntags:\n  - jdex\n---\n\n# Books\n\n" + longDesc + "\n"
		jdexFile := filepath.Join(folderPath, folderName+".md")
		if err := os.WriteFile(jdexFile, []byte(jdexContent), 0o644); err != nil {
			t.Fatal(err)
		}

		desc := readJDexDescription(folderPath)
		if len(desc) != 100 {
			t.Errorf("expected description truncated to 100 chars, got %d chars", len(desc))
		}
	})
}

func TestFormatTreeForSearch_WithJDexDescription(t *testing.T) {
	dir := t.TempDir()
	folderName := "S01.11.11 Theatre"
	folderPath := filepath.Join(dir, folderName)
	if err := os.Mkdir(folderPath, 0o755); err != nil {
		t.Fatal(err)
	}

	jdexContent := "---\ntags:\n  - jdex\n---\n\n# Theatre\n\nShows and performances.\n"
	jdexFile := filepath.Join(folderPath, folderName+".md")
	if err := os.WriteFile(jdexFile, []byte(jdexContent), 0o644); err != nil {
		t.Fatal(err)
	}

	root := &domain.TreeNode{
		Type: domain.IDTypeUnknown,
		ID:   "",
		Name: "Root",
		Children: []*domain.TreeNode{
			{
				Type: domain.IDTypeItem,
				ID:   "S01.11.11",
				Name: "Theatre",
				Path: folderPath,
			},
		},
	}

	result := FormatTreeForSearch(root)
	expected := "S01.11.11 Theatre — Shows and performances.\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

// contains is defined in smartcatalog_test.go
