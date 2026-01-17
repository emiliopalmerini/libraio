package ports

import "libraio/internal/domain"

// VaultIndex provides cached access to vault structure and link graph.
// All query operations should be O(1) or O(log n) via database indexes.
type VaultIndex interface {
	// Lifecycle
	Open(vaultPath string) error
	Close() error

	// Sync operations
	NeedsFullRebuild() bool
	SyncIncremental() (*domain.SyncStats, error)
	SyncFull() (*domain.SyncStats, error)

	// Node queries
	GetNode(path string) (*domain.IndexNode, error)
	GetNodeByJDID(jdID string) (*domain.IndexNode, error)
	GetNextAvailableItemID(categoryID string) (int, error)
	GetNextAvailableCategoryID(areaID string) (int, error)

	// Edge queries (link graph)
	FindLinksToID(targetJDID string) ([]domain.Edge, error)
	FindLinksFromFile(sourcePath string) ([]domain.Edge, error)

	// Batch updates (for move/archive operations)
	BeginTx() (IndexTx, error)
}

// IndexTx represents a transaction for atomic cache updates
type IndexTx interface {
	// Node operations
	UpsertNode(node *domain.IndexNode) error
	DeleteNode(path string) error
	RenameNode(oldPath, newPath string) error

	// Edge operations
	DeleteEdgesFromFile(sourcePath string) error
	InsertEdge(edge *domain.Edge) error
	UpdateEdgeTarget(oldTargetJDID, newTargetJDID, newLinkText string) error

	// Transaction control
	Commit() error
	Rollback() error
}
