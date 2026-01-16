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
