package application

import (
	"fmt"
	"strings"

	"libraio/internal/domain"
)

// ValidateRequired checks if a string field is non-empty (after trimming whitespace).
// Returns a ValidationError if the field is empty.
func ValidateRequired(fieldName, value string) error {
	if strings.TrimSpace(value) == "" {
		// Format field name with spaces for error message (e.g., "categoryID" -> "category ID")
		displayName := formatFieldName(fieldName)
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s is required", displayName),
		}
	}
	return nil
}

// formatFieldName converts camelCase field names to space-separated words
// for more readable error messages (e.g., "categoryID" -> "category ID")
func formatFieldName(fieldName string) string {
	// Handle common patterns directly
	replacements := map[string]string{
		"categoryID":    "category ID",
		"areaID":        "area ID",
		"scopeID":       "scope ID",
		"itemID":        "item ID",
		"description":   "description",
		"sourceID":      "source ID",
		"destinationID": "destination ID",
	}

	if formatted, ok := replacements[fieldName]; ok {
		return formatted
	}

	// Fallback: just return the field name as-is
	return fieldName
}

// ValidateIDType checks if an ID matches the expected Johnny Decimal type.
// Returns a ValidationError if the ID type doesn't match.
func ValidateIDType(fieldName, id string, expectedType domain.IDType) error {
	actualType := domain.ParseIDType(id)
	if actualType != expectedType {
		displayName := formatFieldName(fieldName)
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("expected %s, got: %s", displayName, id),
		}
	}
	return nil
}
