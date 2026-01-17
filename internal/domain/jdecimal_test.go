package domain

import (
	"testing"
)

func TestNextItemID_SkipsStandardZeros(t *testing.T) {
	category := "S01.11"

	// Include standard zeros in existing items - they should be ignored
	existingItems := []string{
		"S01.11.00", // JDex
		"S01.11.01", // Inbox
		"S01.11.09", // Archive
	}

	nextID, err := NextItemID(category, existingItems)
	if err != nil {
		t.Fatalf("NextItemID failed: %v", err)
	}

	// Should return .11, not be affected by standard zeros
	expected := "S01.11.11"
	if nextID != expected {
		t.Errorf("expected %s, got %s", expected, nextID)
	}
}

func TestNextItemID_StartsAt11(t *testing.T) {
	category := "S01.11"

	nextID, err := NextItemID(category, nil)
	if err != nil {
		t.Fatalf("NextItemID failed: %v", err)
	}

	expected := "S01.11.11"
	if nextID != expected {
		t.Errorf("expected %s, got %s", expected, nextID)
	}
}

func TestNextItemID_SkipsUsedIDs(t *testing.T) {
	category := "S01.11"

	existingItems := []string{
		"S01.11.11",
		"S01.11.12",
	}

	nextID, err := NextItemID(category, existingItems)
	if err != nil {
		t.Fatalf("NextItemID failed: %v", err)
	}

	expected := "S01.11.13"
	if nextID != expected {
		t.Errorf("expected %s, got %s", expected, nextID)
	}
}

func TestNextItemID_MixedWithStandardZeros(t *testing.T) {
	category := "S01.11"

	// Mix of standard zeros and regular items
	existingItems := []string{
		"S01.11.00", // Standard zero - should be ignored
		"S01.11.01", // Standard zero - should be ignored
		"S01.11.11", // Regular item - should count
		"S01.11.12", // Regular item - should count
	}

	nextID, err := NextItemID(category, existingItems)
	if err != nil {
		t.Fatalf("NextItemID failed: %v", err)
	}

	expected := "S01.11.13"
	if nextID != expected {
		t.Errorf("expected %s, got %s", expected, nextID)
	}
}

func TestNextItemID_NeverReturnsReservedRange(t *testing.T) {
	category := "S01.11"

	// Even with nothing existing, should never return .00-.10
	for i := 0; i < 100; i++ {
		var existingItems []string
		for j := 11; j < 11+i && j <= 99; j++ {
			existingItems = append(existingItems, "S01.11."+padNum(j))
		}

		nextID, err := NextItemID(category, existingItems)
		if err != nil {
			if i >= 89 { // All IDs exhausted
				continue
			}
			t.Fatalf("NextItemID failed unexpectedly: %v", err)
		}

		num, _ := ExtractNumber(nextID)
		if num <= StandardZeroMax {
			t.Errorf("NextItemID returned reserved ID %s", nextID)
		}
		if num == 10 {
			t.Errorf("NextItemID returned buffer ID .10: %s", nextID)
		}
	}
}

func TestStandardZeroConstants(t *testing.T) {
	if StandardZeroMax != 9 {
		t.Errorf("StandardZeroMax should be 9, got %d", StandardZeroMax)
	}

	if ItemIDStart != 11 {
		t.Errorf("ItemIDStart should be 11, got %d", ItemIDStart)
	}

	// Verify all defined standard zeros are within range
	for _, sz := range StandardZeros {
		if sz.Number > StandardZeroMax {
			t.Errorf("StandardZero %s has number %d > StandardZeroMax %d",
				sz.Name, sz.Number, StandardZeroMax)
		}
	}
}

func padNum(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func TestArchiveItemID(t *testing.T) {
	tests := []struct {
		categoryID string
		want       string
		wantErr    bool
	}{
		{"S01.11", "S01.11.09", false},
		{"S01.21", "S01.21.09", false},
		{"S01.10", "S01.10.09", false},
		{"S01", "", true},       // Scope, not category
		{"S01.10-19", "", true}, // Area, not category
		{"S01.11.11", "", true}, // Item, not category
	}

	for _, tt := range tests {
		t.Run(tt.categoryID, func(t *testing.T) {
			got, err := ArchiveItemID(tt.categoryID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ArchiveItemID(%q) error = %v, wantErr %v", tt.categoryID, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ArchiveItemID(%q) = %q, want %q", tt.categoryID, got, tt.want)
			}
		})
	}
}

func TestIsArchiveItem(t *testing.T) {
	tests := []struct {
		itemID string
		want   bool
	}{
		{"S01.11.09", true},  // Archive item
		{"S01.21.09", true},  // Archive item
		{"S01.11.01", false}, // Inbox, not archive
		{"S01.11.11", false}, // Regular item
		{"S01.11", false},    // Category, not item
		{"S01", false},       // Scope, not item
	}

	for _, tt := range tests {
		t.Run(tt.itemID, func(t *testing.T) {
			got := IsArchiveItem(tt.itemID)
			if got != tt.want {
				t.Errorf("IsArchiveItem(%q) = %v, want %v", tt.itemID, got, tt.want)
			}
		})
	}
}

func TestIsAreaManagementCategory(t *testing.T) {
	tests := []struct {
		categoryID string
		want       bool
	}{
		{"S01.10", true},     // Area management category
		{"S01.20", true},     // Area management category
		{"S01.30", true},     // Area management category
		{"S01.11", false},    // Regular category
		{"S01.21", false},    // Regular category
		{"S01.19", false},    // Archive category
		{"S01.29", false},    // Archive category
		{"S01", false},       // Scope, not category
		{"S01.10-19", false}, // Area, not category
		{"S01.11.11", false}, // Item, not category
	}

	for _, tt := range tests {
		t.Run(tt.categoryID, func(t *testing.T) {
			got := IsAreaManagementCategory(tt.categoryID)
			if got != tt.want {
				t.Errorf("IsAreaManagementCategory(%q) = %v, want %v", tt.categoryID, got, tt.want)
			}
		})
	}
}

func TestAreaRangeFromCategory(t *testing.T) {
	tests := []struct {
		categoryID string
		want       string
	}{
		{"S01.10", "10-19"},
		{"S01.11", "10-19"},
		{"S01.19", "10-19"},
		{"S01.20", "20-29"},
		{"S01.25", "20-29"},
		{"S01.30", "30-39"},
		{"S01", ""},       // Scope, not category
		{"S01.10-19", ""}, // Area, not category
		{"S01.11.11", ""}, // Item, not category
	}

	for _, tt := range tests {
		t.Run(tt.categoryID, func(t *testing.T) {
			got := AreaRangeFromCategory(tt.categoryID)
			if got != tt.want {
				t.Errorf("AreaRangeFromCategory(%q) = %q, want %q", tt.categoryID, got, tt.want)
			}
		})
	}
}

func TestStandardZeroNameForContext(t *testing.T) {
	tests := []struct {
		baseName   string
		categoryID string
		want       string
	}{
		// Area management categories (.X0) get "for SXX.X0-X9" suffix
		{"Inbox", "S01.10", "Inbox for S01.10-19"},
		{"Tasks", "S01.20", "Tasks for S01.20-29"},
		{"Archive", "S01.30", "Archive for S01.30-39"},
		// Regular categories get "for SXX.XX" suffix
		{"Inbox", "S01.11", "Inbox for S01.11"},
		{"Tasks", "S01.21", "Tasks for S01.21"},
		{"Archive", "S01.19", "Archive for S01.19"},
		// Different scopes
		{"Inbox", "S02.10", "Inbox for S02.10-19"},
		{"Inbox", "S02.11", "Inbox for S02.11"},
		{"Inbox", "S03.25", "Inbox for S03.25"},
	}

	for _, tt := range tests {
		t.Run(tt.baseName+"_"+tt.categoryID, func(t *testing.T) {
			got := StandardZeroNameForContext(tt.baseName, tt.categoryID)
			if got != tt.want {
				t.Errorf("StandardZeroNameForContext(%q, %q) = %q, want %q", tt.baseName, tt.categoryID, got, tt.want)
			}
		})
	}
}

func TestAreaArchiveItemID(t *testing.T) {
	tests := []struct {
		categoryID string
		want       string
		wantErr    bool
	}{
		// Regular categories → area's .X0.09 archive
		{"S01.11", "S01.10.09", false},
		{"S01.12", "S01.10.09", false},
		{"S01.19", "S01.10.09", false},
		{"S01.21", "S01.20.09", false},
		{"S01.25", "S01.20.09", false},
		{"S02.35", "S02.30.09", false},
		// Management categories (.X0) should error - can't archive to self
		{"S01.10", "", true},
		{"S01.20", "", true},
		// Invalid inputs
		{"S01", "", true},       // Scope
		{"S01.10-19", "", true}, // Area
		{"S01.11.11", "", true}, // Item
	}

	for _, tt := range tests {
		t.Run(tt.categoryID, func(t *testing.T) {
			got, err := AreaArchiveItemID(tt.categoryID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AreaArchiveItemID(%q) error = %v, wantErr %v", tt.categoryID, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AreaArchiveItemID(%q) = %q, want %q", tt.categoryID, got, tt.want)
			}
		})
	}
}

func TestManagementCategoryID(t *testing.T) {
	tests := []struct {
		categoryID string
		want       string
		wantErr    bool
	}{
		// Regular categories → their area's management category (.X0)
		{"S01.11", "S01.10", false},
		{"S01.19", "S01.10", false},
		{"S01.21", "S01.20", false},
		{"S01.35", "S01.30", false},
		{"S02.45", "S02.40", false},
		// Management categories return themselves
		{"S01.10", "S01.10", false},
		{"S01.20", "S01.20", false},
		// Invalid inputs
		{"S01", "", true},       // Scope
		{"S01.10-19", "", true}, // Area
		{"S01.11.11", "", true}, // Item
	}

	for _, tt := range tests {
		t.Run(tt.categoryID, func(t *testing.T) {
			got, err := ManagementCategoryID(tt.categoryID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ManagementCategoryID(%q) error = %v, wantErr %v", tt.categoryID, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ManagementCategoryID(%q) = %q, want %q", tt.categoryID, got, tt.want)
			}
		})
	}
}

func TestJDexFileName(t *testing.T) {
	tests := []struct {
		folderName string
		want       string
	}{
		{"S01.11.01 Inbox for S01.11", "S01.11.01 Inbox for S01.11.md"},
		{"S01.10.01 Inbox for S01.10-19", "S01.10.01 Inbox for S01.10-19.md"},
		{"S01.11.11 Theatre", "S01.11.11 Theatre.md"},
	}

	for _, tt := range tests {
		t.Run(tt.folderName, func(t *testing.T) {
			got := JDexFileName(tt.folderName)
			if got != tt.want {
				t.Errorf("JDexFileName(%q) = %q, want %q", tt.folderName, got, tt.want)
			}
		})
	}
}
