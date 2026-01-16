package domain

import (
	"fmt"
	"strings"
	"time"
)

// ReadmeTemplate generates a README.md content for a new item
func ReadmeTemplate(id, description string) string {
	now := time.Now()
	dateStr := now.Format("2006/01/02")

	fullTitle := fmt.Sprintf("%s %s", id, description)

	return fmt.Sprintf(`---
aliases:
  - %s
created: %s
location: Obsidian
tags:
  - jdex
  - index
---

# %s

%s.
`, fullTitle, dateStr, fullTitle, formatDescriptionSentence(description))
}

// formatDescriptionSentence creates a descriptive sentence from the description
func formatDescriptionSentence(description string) string {
	if description == "" {
		return "Description pending"
	}

	// Capitalize first letter
	desc := strings.ToUpper(description[:1]) + description[1:]

	// Remove trailing period if present (we'll add our own)
	desc = strings.TrimSuffix(desc, ".")

	return desc
}

// StandardZeroReadmeTemplate generates a README.md content for a standard zero item
func StandardZeroReadmeTemplate(id, name, purpose string) string {
	now := time.Now()
	dateStr := now.Format("2006/01/02")

	fullTitle := fmt.Sprintf("%s %s", id, name)

	return fmt.Sprintf(`---
aliases:
  - %s
created: %s
location: Obsidian
tags:
  - jdex
  - index
  - standard-zero
---

# %s

%s
`, fullTitle, dateStr, fullTitle, purpose)
}

// ParseReadmeFrontmatter extracts frontmatter fields from README content
type ReadmeFrontmatter struct {
	Aliases  []string
	Created  string
	Location string
	Tags     []string
}

// UpdateReadmeID updates the ID in an existing README content
// This is useful when moving or archiving items
func UpdateReadmeID(content, oldID, newID, newDescription string) string {
	oldTitle := fmt.Sprintf("%s %s", oldID, ExtractDescription(oldID+" "+newDescription))
	newTitle := fmt.Sprintf("%s %s", newID, newDescription)

	// Update alias
	content = strings.Replace(content, oldID, newID, -1)

	// Update title in header
	content = strings.Replace(content, oldTitle, newTitle, -1)

	return content
}
