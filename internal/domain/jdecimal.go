package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// IDType represents the type of Johnny Decimal identifier
type IDType int

const (
	IDTypeUnknown  IDType = iota
	IDTypeScope           // S00, S01, S02, S03
	IDTypeArea            // S01.10-19
	IDTypeCategory        // S01.11
	IDTypeItem            // S01.11.11
)

func (t IDType) String() string {
	switch t {
	case IDTypeScope:
		return "Scope"
	case IDTypeArea:
		return "Area"
	case IDTypeCategory:
		return "Category"
	case IDTypeItem:
		return "Item"
	default:
		return "Unknown"
	}
}

var (
	scopeRegex    = regexp.MustCompile(`^S0[0-9]$`)
	areaRegex     = regexp.MustCompile(`^S0[0-9]\.[0-9]0-[0-9]9$`)
	categoryRegex = regexp.MustCompile(`^S0[0-9]\.[0-9][0-9]$`)
	itemRegex     = regexp.MustCompile(`^S0[0-9]\.[0-9][0-9]\.[0-9][0-9]$`)
)

// StandardZero represents a standard zero item definition
type StandardZero struct {
	Number  int
	Name    string
	Purpose string
}

// StandardZeros defines the reserved IDs (.00-.09) for management items
var StandardZeros = []StandardZero{
	{0, "JDex", "Index and metadata for this category. Use this to track what IDs exist and their purposes."},
	{1, "Inbox", "Temporary landing zone for items that need to be sorted or processed."},
	{2, "Tasks", "Active tasks and projects related to this category."},
	{3, "Templates", "Reusable templates and boilerplate for creating new items."},
	{4, "Links", "External references, bookmarks, and related resources."},
	{8, "Someday", "Items to revisit in the future when time permits."},
	{9, "Archive", "Inactive or completed items preserved for reference."},
}

const (
	// StandardZeroMax is the highest reserved ID for standard zeros (.00-.09)
	StandardZeroMax = 9
	// ItemIDStart is the first ID available for regular items (.11+, skipping .10 as buffer)
	ItemIDStart = 11
)

// ParseIDType determines the type of a Johnny Decimal ID string
func ParseIDType(id string) IDType {
	id = strings.TrimSpace(id)

	switch {
	case scopeRegex.MatchString(id):
		return IDTypeScope
	case areaRegex.MatchString(id):
		return IDTypeArea
	case categoryRegex.MatchString(id):
		return IDTypeCategory
	case itemRegex.MatchString(id):
		return IDTypeItem
	default:
		return IDTypeUnknown
	}
}

// ValidateID checks if a string is a valid Johnny Decimal ID
func ValidateID(id string) error {
	if ParseIDType(id) == IDTypeUnknown {
		return fmt.Errorf("invalid Johnny Decimal ID: %s", id)
	}
	return nil
}

// ParseScope extracts the scope from any valid ID
func ParseScope(id string) (string, error) {
	if len(id) < 3 {
		return "", fmt.Errorf("ID too short: %s", id)
	}
	scope := id[:3]
	if !scopeRegex.MatchString(scope) {
		return "", fmt.Errorf("invalid scope in ID: %s", id)
	}
	return scope, nil
}

// ParseArea extracts the area range from a category or item ID
// Returns the area in format "SXX.X0-X9"
func ParseArea(id string) (string, error) {
	idType := ParseIDType(id)
	if idType != IDTypeCategory && idType != IDTypeItem {
		return "", fmt.Errorf("cannot extract area from %s type: %s", idType, id)
	}

	// Extract the tens digit (e.g., S01.11 -> 1, S01.11.11 -> 1)
	parts := strings.Split(id, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid ID format: %s", id)
	}

	categoryNum := parts[1][:2]
	tensDigit := categoryNum[0:1]

	return fmt.Sprintf("%s.%s0-%s9", parts[0], tensDigit, tensDigit), nil
}

// ParseCategory extracts the category from an item ID
func ParseCategory(id string) (string, error) {
	if ParseIDType(id) != IDTypeItem {
		return "", fmt.Errorf("cannot extract category from non-item ID: %s", id)
	}

	parts := strings.Split(id, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid item ID format: %s", id)
	}

	return fmt.Sprintf("%s.%s", parts[0], parts[1]), nil
}

// ExtractNumber extracts the numeric portion of an ID based on its type
func ExtractNumber(id string) (int, error) {
	idType := ParseIDType(id)

	switch idType {
	case IDTypeScope:
		n, err := strconv.Atoi(id[2:3])
		return n, err
	case IDTypeCategory:
		parts := strings.Split(id, ".")
		n, err := strconv.Atoi(parts[1])
		return n, err
	case IDTypeItem:
		parts := strings.Split(id, ".")
		n, err := strconv.Atoi(parts[2])
		return n, err
	default:
		return 0, fmt.Errorf("cannot extract number from %s: %s", idType, id)
	}
}

// NextCategoryID generates the next category ID within an area
func NextCategoryID(area string, existingCategories []string) (string, error) {
	if ParseIDType(area) != IDTypeArea {
		return "", fmt.Errorf("invalid area ID: %s", area)
	}

	// Parse area bounds (e.g., S01.10-19 -> min=10, max=18, archive=19)
	parts := strings.Split(area, ".")
	scope := parts[0]
	rangePart := parts[1] // e.g., "10-19"
	rangeParts := strings.Split(rangePart, "-")

	minNum, _ := strconv.Atoi(rangeParts[0])
	maxNum, _ := strconv.Atoi(rangeParts[1])

	// Find existing category numbers
	used := make(map[int]bool)
	for _, cat := range existingCategories {
		if num, err := ExtractNumber(cat); err == nil {
			used[num] = true
		}
	}

	// Find next available (skip X0 for index and X9 for archive)
	for i := minNum + 1; i < maxNum; i++ {
		if !used[i] {
			return fmt.Sprintf("%s.%02d", scope, i), nil
		}
	}

	return "", fmt.Errorf("no available category IDs in area %s", area)
}

// NextItemID generates the next item ID within a category.
// IDs .00-.09 are reserved for standard zeros and are never returned.
// Regular item IDs start at .11 (with .10 skipped as a buffer).
func NextItemID(category string, existingItems []string) (string, error) {
	if ParseIDType(category) != IDTypeCategory {
		return "", fmt.Errorf("invalid category ID: %s", category)
	}

	// Find existing item numbers, excluding standard zeros (.00-.09)
	used := make(map[int]bool)
	for _, item := range existingItems {
		if num, err := ExtractNumber(item); err == nil {
			// Skip standard zeros - they're managed separately
			if num <= StandardZeroMax {
				continue
			}
			used[num] = true
		}
	}

	// Start from ItemIDStart (.11), never use reserved range
	for i := ItemIDStart; i <= 99; i++ {
		if !used[i] {
			return fmt.Sprintf("%s.%02d", category, i), nil
		}
	}

	return "", fmt.Errorf("no available item IDs in category %s", category)
}

// ArchiveCategory returns the archive category for an area
func ArchiveCategory(area string) (string, error) {
	if ParseIDType(area) != IDTypeArea {
		return "", fmt.Errorf("invalid area ID: %s", area)
	}

	parts := strings.Split(area, ".")
	scope := parts[0]
	rangePart := parts[1]
	rangeParts := strings.Split(rangePart, "-")

	archiveNum, _ := strconv.Atoi(rangeParts[1])
	return fmt.Sprintf("%s.%02d", scope, archiveNum), nil
}

// IsAreaManagementCategory checks if a category ID is an area-level management category (.X0)
// e.g., S01.10 -> true, S01.11 -> false, S01.20 -> true
func IsAreaManagementCategory(categoryID string) bool {
	if ParseIDType(categoryID) != IDTypeCategory {
		return false
	}
	// Category ID format: S0X.YZ - check if Z is 0
	parts := strings.Split(categoryID, ".")
	if len(parts) != 2 || len(parts[1]) != 2 {
		return false
	}
	return parts[1][1] == '0'
}

// AreaRangeFromCategory returns the area range string for a category
// e.g., S01.10 -> "10-19", S01.21 -> "20-29"
func AreaRangeFromCategory(categoryID string) string {
	if ParseIDType(categoryID) != IDTypeCategory {
		return ""
	}
	parts := strings.Split(categoryID, ".")
	if len(parts) != 2 || len(parts[1]) != 2 {
		return ""
	}
	tensDigit := parts[1][0:1]
	return fmt.Sprintf("%s0-%s9", tensDigit, tensDigit)
}

// StandardZeroNameForContext returns the appropriate standard zero name based on category context
// For area-level categories (.X0): "Inbox" -> "Inbox for area X0-X9"
// For regular categories: "Inbox" -> "Inbox"
func StandardZeroNameForContext(baseName, categoryID string) string {
	if IsAreaManagementCategory(categoryID) {
		areaRange := AreaRangeFromCategory(categoryID)
		return fmt.Sprintf("%s for area %s", baseName, areaRange)
	}
	return baseName
}

// JDexFileName returns the JDex filename for a folder name
// e.g., "S01.11.01 Inbox" -> "S01.11.01 Inbox.md"
func JDexFileName(folderName string) string {
	return folderName + ".md"
}

// ExtractDescription extracts the description from a folder name
// e.g., "S01.11.15 Theatre" -> "Theatre"
func ExtractDescription(folderName string) string {
	parts := strings.SplitN(folderName, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// ExtractID extracts the ID from a folder name
// e.g., "S01.11.15 Theatre" -> "S01.11.15"
func ExtractID(folderName string) string {
	parts := strings.SplitN(folderName, " ", 2)
	return parts[0]
}

// FormatFolderName creates a folder name from ID and description
func FormatFolderName(id, description string) string {
	return fmt.Sprintf("%s %s", id, description)
}
