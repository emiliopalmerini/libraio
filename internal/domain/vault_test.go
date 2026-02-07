package domain

import (
	"testing"
)

func TestSortByID(t *testing.T) {
	t.Run("sorts scopes by ID", func(t *testing.T) {
		scopes := []Scope{
			{ID: "S03", Name: "Third"},
			{ID: "S01", Name: "First"},
			{ID: "S02", Name: "Second"},
		}

		SortByID(scopes)

		if scopes[0].ID != "S01" {
			t.Errorf("expected first scope to be S01, got %s", scopes[0].ID)
		}
		if scopes[1].ID != "S02" {
			t.Errorf("expected second scope to be S02, got %s", scopes[1].ID)
		}
		if scopes[2].ID != "S03" {
			t.Errorf("expected third scope to be S03, got %s", scopes[2].ID)
		}
	})

	t.Run("sorts areas by ID", func(t *testing.T) {
		areas := []Area{
			{ID: "S01.30-39", Name: "Third"},
			{ID: "S01.10-19", Name: "First"},
			{ID: "S01.20-29", Name: "Second"},
		}

		SortByID(areas)

		if areas[0].ID != "S01.10-19" {
			t.Errorf("expected first area to be S01.10-19, got %s", areas[0].ID)
		}
		if areas[1].ID != "S01.20-29" {
			t.Errorf("expected second area to be S01.20-29, got %s", areas[1].ID)
		}
		if areas[2].ID != "S01.30-39" {
			t.Errorf("expected third area to be S01.30-39, got %s", areas[2].ID)
		}
	})

	t.Run("sorts categories by ID", func(t *testing.T) {
		categories := []Category{
			{ID: "S01.13", Name: "Third"},
			{ID: "S01.11", Name: "First"},
			{ID: "S01.12", Name: "Second"},
		}

		SortByID(categories)

		if categories[0].ID != "S01.11" {
			t.Errorf("expected first category to be S01.11, got %s", categories[0].ID)
		}
		if categories[1].ID != "S01.12" {
			t.Errorf("expected second category to be S01.12, got %s", categories[1].ID)
		}
		if categories[2].ID != "S01.13" {
			t.Errorf("expected third category to be S01.13, got %s", categories[2].ID)
		}
	})

	t.Run("sorts items by ID", func(t *testing.T) {
		items := []Item{
			{ID: "S01.11.15", Name: "Third"},
			{ID: "S01.11.11", Name: "First"},
			{ID: "S01.11.13", Name: "Second"},
		}

		SortByID(items)

		if items[0].ID != "S01.11.11" {
			t.Errorf("expected first item to be S01.11.11, got %s", items[0].ID)
		}
		if items[1].ID != "S01.11.13" {
			t.Errorf("expected second item to be S01.11.13, got %s", items[1].ID)
		}
		if items[2].ID != "S01.11.15" {
			t.Errorf("expected third item to be S01.11.15, got %s", items[2].ID)
		}
	})

	t.Run("handles empty slices", func(t *testing.T) {
		scopes := []Scope{}
		SortByID(scopes)
		if len(scopes) != 0 {
			t.Errorf("expected empty slice to remain empty")
		}
	})

	t.Run("handles single element", func(t *testing.T) {
		scopes := []Scope{{ID: "S01", Name: "Only"}}
		SortByID(scopes)
		if len(scopes) != 1 || scopes[0].ID != "S01" {
			t.Errorf("expected single element to remain unchanged")
		}
	})
}

// Test that old functions still work (for backwards compatibility during transition)
func TestLegacySortFunctions(t *testing.T) {
	t.Run("SortScopes", func(t *testing.T) {
		scopes := []Scope{
			{ID: "S03", Name: "Third"},
			{ID: "S01", Name: "First"},
			{ID: "S02", Name: "Second"},
		}

		SortScopes(scopes)

		if scopes[0].ID != "S01" || scopes[1].ID != "S02" || scopes[2].ID != "S03" {
			t.Errorf("SortScopes failed to sort correctly")
		}
	})

	t.Run("SortAreas", func(t *testing.T) {
		areas := []Area{
			{ID: "S01.30-39", Name: "Third"},
			{ID: "S01.10-19", Name: "First"},
		}

		SortAreas(areas)

		if areas[0].ID != "S01.10-19" {
			t.Errorf("SortAreas failed to sort correctly")
		}
	})

	t.Run("SortCategories", func(t *testing.T) {
		categories := []Category{
			{ID: "S01.13", Name: "Third"},
			{ID: "S01.11", Name: "First"},
		}

		SortCategories(categories)

		if categories[0].ID != "S01.11" {
			t.Errorf("SortCategories failed to sort correctly")
		}
	})

	t.Run("SortItems", func(t *testing.T) {
		items := []Item{
			{ID: "S01.11.15", Name: "Third"},
			{ID: "S01.11.11", Name: "First"},
		}

		SortItems(items)

		if items[0].ID != "S01.11.11" {
			t.Errorf("SortItems failed to sort correctly")
		}
	})
}
