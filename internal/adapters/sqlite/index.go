package sqlite

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"libraio/internal/domain"
	"libraio/internal/ports"

	_ "modernc.org/sqlite"
)

const schemaVersion = "1"

// Index implements ports.VaultIndex using SQLite
type Index struct {
	db        *sql.DB
	vaultPath string
	dbPath    string
}

// Ensure Index implements VaultIndex
var _ ports.VaultIndex = (*Index)(nil)

// NewIndex creates a new SQLite index
func NewIndex() *Index {
	return &Index{}
}

// Open initializes the index for the given vault path
func (idx *Index) Open(vaultPath string) error {
	// Expand ~ in path
	if len(vaultPath) > 0 && vaultPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		vaultPath = filepath.Join(home, vaultPath[1:])
	}

	idx.vaultPath = vaultPath
	idx.dbPath = databasePath(vaultPath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(idx.dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite", idx.dbPath+"?_journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	idx.db = db

	// Performance pragmas + schema in single batch (reduces round-trips)
	_, err = db.Exec(`
		PRAGMA synchronous = NORMAL;
		PRAGMA cache_size = -64000;
		PRAGMA temp_store = MEMORY;
		PRAGMA busy_timeout = 5000;

		CREATE TABLE IF NOT EXISTS nodes (
			path TEXT PRIMARY KEY,
			jd_id TEXT,
			jd_type TEXT,
			name TEXT,
			mtime INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS edges (
			source_path TEXT NOT NULL,
			target_jd_id TEXT NOT NULL,
			link_text TEXT NOT NULL,
			PRIMARY KEY (source_path, link_text)
		);
		CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_nodes_jd_id ON nodes(jd_id);
		CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_jd_id);
		CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_path);
	`)
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to setup database: %w", err)
	}

	// Update metadata
	if err := idx.updateMeta(); err != nil {
		db.Close()
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

// Close closes the database connection
func (idx *Index) Close() error {
	if idx.db != nil {
		return idx.db.Close()
	}
	return nil
}

// NeedsFullRebuild returns true if the index should be fully rebuilt
func (idx *Index) NeedsFullRebuild() bool {
	var version, vaultHash string

	idx.db.QueryRow("SELECT value FROM meta WHERE key = 'schema_version'").Scan(&version)
	idx.db.QueryRow("SELECT value FROM meta WHERE key = 'vault_path_hash'").Scan(&vaultHash)

	expectedHash := hashVaultPath(idx.vaultPath)

	return version != schemaVersion || vaultHash != expectedHash
}

// databasePath returns the path for the SQLite database
func databasePath(vaultPath string) string {
	// XDG data directory
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}

	// Hash vault path for unique DB name
	hash := hashVaultPath(vaultPath)

	return filepath.Join(dataHome, "libraio", hash+".db")
}

// hashVaultPath returns a short hash of the vault path
func hashVaultPath(vaultPath string) string {
	h := sha256.Sum256([]byte(vaultPath))
	return hex.EncodeToString(h[:8]) // First 8 bytes = 16 hex chars
}

// updateMeta updates the schema version and vault path hash
func (idx *Index) updateMeta() error {
	_, err := idx.db.Exec(`
		INSERT OR REPLACE INTO meta (key, value) VALUES ('schema_version', ?);
		INSERT OR REPLACE INTO meta (key, value) VALUES ('vault_path_hash', ?);
	`, schemaVersion, hashVaultPath(idx.vaultPath))
	return err
}

// GetNode retrieves a node by path
func (idx *Index) GetNode(path string) (*domain.IndexNode, error) {
	var node domain.IndexNode
	var jdType sql.NullString

	err := idx.db.QueryRow(`
		SELECT path, jd_id, jd_type, name, mtime
		FROM nodes WHERE path = ?
	`, path).Scan(&node.Path, &node.JDID, &jdType, &node.Name, &node.Mtime)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if jdType.Valid {
		node.JDType = domain.ParseIDType(jdType.String)
	}

	return &node, nil
}

// GetNodeByJDID retrieves a node by Johnny Decimal ID
func (idx *Index) GetNodeByJDID(jdID string) (*domain.IndexNode, error) {
	var node domain.IndexNode
	var jdType sql.NullString

	err := idx.db.QueryRow(`
		SELECT path, jd_id, jd_type, name, mtime
		FROM nodes WHERE jd_id = ?
	`, jdID).Scan(&node.Path, &node.JDID, &jdType, &node.Name, &node.Mtime)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if jdType.Valid {
		node.JDType = domain.ParseIDType(jdType.String)
	}

	return &node, nil
}

// GetNextAvailableItemID returns the next available item number for a category
func (idx *Index) GetNextAvailableItemID(categoryID string) (int, error) {
	var maxID sql.NullInt64

	// Pattern: categoryID.XX where XX is the item number
	pattern := categoryID + ".%"

	err := idx.db.QueryRow(`
		SELECT MAX(CAST(SUBSTR(jd_id, -2) AS INTEGER))
		FROM nodes
		WHERE jd_id LIKE ? AND jd_type = 'item'
	`, pattern).Scan(&maxID)

	if err != nil {
		return 11, err // Start at .11 for regular content
	}

	if !maxID.Valid || maxID.Int64 < 11 {
		return 11, nil // Start at .11 for regular content
	}

	return int(maxID.Int64) + 1, nil
}

// GetNextAvailableCategoryID returns the next available category number for an area
func (idx *Index) GetNextAvailableCategoryID(areaID string) (int, error) {
	var maxID sql.NullInt64

	// Extract scope and area start (e.g., "S01.10-19" -> "S01.1")
	// Categories in area 10-19 are 10, 11, 12, ..., 19
	// We need to find the max category number in this range

	// For simplicity, find all categories with this area prefix
	// Pattern: S01.1X where X is 0-9
	// This assumes area ID format is "S01.10-19"
	if len(areaID) < 5 {
		return 11, fmt.Errorf("invalid area ID: %s", areaID)
	}

	// Extract scope.areaPrefix (e.g., "S01.1" from "S01.10-19")
	scopeAreaPrefix := areaID[:5] // "S01.1"
	pattern := scopeAreaPrefix + "%"

	err := idx.db.QueryRow(`
		SELECT MAX(CAST(SUBSTR(jd_id, 5, 2) AS INTEGER))
		FROM nodes
		WHERE jd_id LIKE ? AND jd_type = 'category'
	`, pattern).Scan(&maxID)

	if err != nil {
		return 11, err
	}

	if !maxID.Valid || maxID.Int64 < 11 {
		return 11, nil // Start at X1 (e.g., 11, 21, 31...)
	}

	return int(maxID.Int64) + 1, nil
}

// FindLinksToID returns all edges pointing to a JD ID
func (idx *Index) FindLinksToID(targetJDID string) ([]domain.Edge, error) {
	rows, err := idx.db.Query(`
		SELECT source_path, target_jd_id, link_text
		FROM edges WHERE target_jd_id = ?
	`, targetJDID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []domain.Edge
	for rows.Next() {
		var e domain.Edge
		if err := rows.Scan(&e.SourcePath, &e.TargetJDID, &e.LinkText); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}

	return edges, rows.Err()
}

// FindLinksFromFile returns all edges from a source file
func (idx *Index) FindLinksFromFile(sourcePath string) ([]domain.Edge, error) {
	rows, err := idx.db.Query(`
		SELECT source_path, target_jd_id, link_text
		FROM edges WHERE source_path = ?
	`, sourcePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []domain.Edge
	for rows.Next() {
		var e domain.Edge
		if err := rows.Scan(&e.SourcePath, &e.TargetJDID, &e.LinkText); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}

	return edges, rows.Err()
}

// BeginTx starts a new transaction
func (idx *Index) BeginTx() (ports.IndexTx, error) {
	tx, err := idx.db.Begin()
	if err != nil {
		return nil, err
	}
	return &indexTx{tx: tx}, nil
}
