package domain

import "strings"

// InboxLevel represents the scope level of an inbox
type InboxLevel int

const (
	InboxLevelCategory InboxLevel = iota // S01.11.01 -> only S01.11 items
	InboxLevelArea                       // S01.10.01 -> all S01.10-19 items
	InboxLevelScope                      // S01.01.01 -> all S01 items
)

// IsInboxItem checks if an item ID is an inbox item (.01)
func IsInboxItem(itemID string) bool {
	if ParseIDType(itemID) != IDTypeItem {
		return false
	}
	parts := strings.Split(itemID, ".")
	return len(parts) == 3 && parts[2] == "01"
}

// GetInboxLevel determines the scope of context for an inbox item
func GetInboxLevel(inboxItemID string) InboxLevel {
	parentCat, err := ParseCategory(inboxItemID)
	if err != nil {
		return InboxLevelCategory
	}

	parts := strings.Split(parentCat, ".")
	if len(parts) != 2 || len(parts[1]) != 2 {
		return InboxLevelCategory
	}
	catNum := parts[1] // e.g., "11", "10", "01"

	// Scope management area: 00-09 (categories 01-09, but 00 is JDex)
	if catNum[0] == '0' && catNum != "00" {
		return InboxLevelScope
	}
	// Area management category: X0 (e.g., 10, 20, 30)
	if catNum[1] == '0' {
		return InboxLevelArea
	}
	return InboxLevelCategory
}

// GetInboxParentID returns the parent scope/area/category ID for context building
func GetInboxParentID(inboxItemID string) (string, InboxLevel) {
	level := GetInboxLevel(inboxItemID)
	switch level {
	case InboxLevelScope:
		scope, _ := ParseScope(inboxItemID)
		return scope, level
	case InboxLevelArea:
		area, _ := ParseArea(inboxItemID)
		return area, level
	default:
		cat, _ := ParseCategory(inboxItemID)
		return cat, level
	}
}
