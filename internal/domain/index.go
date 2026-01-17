package domain

import "time"

// IndexNode represents a cached vault entry (JD node or markdown file)
type IndexNode struct {
	Path   string // Relative path from vault root (primary key)
	JDID   string // Johnny Decimal ID (empty for non-JD files)
	JDType IDType // Scope, Area, Category, Item, or IDTypeUnknown
	Name   string // Description/filename
	Mtime  int64  // Unix timestamp for incremental sync
}

// Edge represents an Obsidian wiki link between files
type Edge struct {
	SourcePath string // File containing the link
	TargetJDID string // Referenced JD ID
	LinkText   string // Original [[link]] text
}

// SyncStats holds statistics from a sync operation
type SyncStats struct {
	NodesAdded   int
	NodesUpdated int
	NodesDeleted int
	EdgesAdded   int
	EdgesDeleted int
	FilesScanned int
	Duration     time.Duration
}
