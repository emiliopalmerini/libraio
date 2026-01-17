package domain

import "testing"

func TestIsInboxItem(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		// Inbox items (should return true)
		{"S01.11.01", true},
		{"S01.10.01", true},
		{"S02.21.01", true},
		{"S03.35.01", true},

		// Non-inbox items (should return false)
		{"S01.11.02", false}, // Tasks
		{"S01.11.09", false}, // Archive
		{"S01.11.11", false}, // Regular item
		{"S01.11.15", false}, // Regular item
		{"S01.11.00", false}, // JDex

		// Non-item IDs (should return false)
		{"S01", false},       // Scope
		{"S01.10-19", false}, // Area
		{"S01.11", false},    // Category

		// Invalid IDs (should return false)
		{"", false},
		{"invalid", false},
		{"S01.11.1", false}, // Invalid format
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := IsInboxItem(tt.id)
			if result != tt.expected {
				t.Errorf("IsInboxItem(%q) = %v, expected %v", tt.id, result, tt.expected)
			}
		})
	}
}

func TestGetInboxLevel(t *testing.T) {
	tests := []struct {
		id       string
		expected InboxLevel
	}{
		// Category inbox (regular categories like S01.11)
		{"S01.11.01", InboxLevelCategory}, // Inbox for S01.11
		{"S01.12.01", InboxLevelCategory}, // Inbox for S01.12
		{"S02.25.01", InboxLevelCategory}, // Inbox for S02.25

		// Area inbox (area management categories like S01.10, S01.20)
		{"S01.10.01", InboxLevelArea}, // Inbox for S01.10-19
		{"S01.20.01", InboxLevelArea}, // Inbox for S01.20-29
		{"S02.30.01", InboxLevelArea}, // Inbox for S02.30-39

		// Scope inbox (scope management area 00-09, categories 01-09)
		{"S01.01.01", InboxLevelScope}, // Inbox for S01.00-09 (scope management)
		{"S01.02.01", InboxLevelScope}, // Tasks for S01.00-09
		{"S02.01.01", InboxLevelScope}, // Inbox for S02.00-09

		// Edge cases - defaults to Category
		{"invalid", InboxLevelCategory},
		{"S01.11", InboxLevelCategory}, // Not an item ID
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := GetInboxLevel(tt.id)
			if result != tt.expected {
				t.Errorf("GetInboxLevel(%q) = %v, expected %v", tt.id, result, tt.expected)
			}
		})
	}
}

func TestGetInboxParentID(t *testing.T) {
	tests := []struct {
		id            string
		expectedID    string
		expectedLevel InboxLevel
	}{
		// Category inbox -> returns category ID
		{"S01.11.01", "S01.11", InboxLevelCategory},
		{"S02.25.01", "S02.25", InboxLevelCategory},

		// Area inbox -> returns area ID
		{"S01.10.01", "S01.10-19", InboxLevelArea},
		{"S01.20.01", "S01.20-29", InboxLevelArea},

		// Scope inbox -> returns scope ID
		{"S01.01.01", "S01", InboxLevelScope},
		{"S02.01.01", "S02", InboxLevelScope},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			parentID, level := GetInboxParentID(tt.id)
			if parentID != tt.expectedID {
				t.Errorf("GetInboxParentID(%q) parentID = %q, expected %q", tt.id, parentID, tt.expectedID)
			}
			if level != tt.expectedLevel {
				t.Errorf("GetInboxParentID(%q) level = %v, expected %v", tt.id, level, tt.expectedLevel)
			}
		})
	}
}
