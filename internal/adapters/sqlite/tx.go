package sqlite

import (
	"database/sql"

	"libraio/internal/domain"
	"libraio/internal/ports"
)

// indexTx implements ports.IndexTx
type indexTx struct {
	tx *sql.Tx
}

// Ensure indexTx implements IndexTx
var _ ports.IndexTx = (*indexTx)(nil)

// UpsertNode inserts or updates a node
func (t *indexTx) UpsertNode(node *domain.IndexNode) error {
	_, err := t.tx.Exec(`
		INSERT OR REPLACE INTO nodes (path, jd_id, jd_type, name, mtime)
		VALUES (?, ?, ?, ?, ?)
	`, node.Path, node.JDID, node.JDType.String(), node.Name, node.Mtime)
	return err
}

// DeleteNode removes a node by path
func (t *indexTx) DeleteNode(path string) error {
	_, err := t.tx.Exec(`DELETE FROM nodes WHERE path = ?`, path)
	return err
}

// RenameNode updates a node's path
func (t *indexTx) RenameNode(oldPath, newPath string) error {
	_, err := t.tx.Exec(`UPDATE nodes SET path = ? WHERE path = ?`, newPath, oldPath)
	return err
}

// DeleteEdgesFromFile removes all edges from a source file
func (t *indexTx) DeleteEdgesFromFile(sourcePath string) error {
	_, err := t.tx.Exec(`DELETE FROM edges WHERE source_path = ?`, sourcePath)
	return err
}

// InsertEdge adds a new edge
func (t *indexTx) InsertEdge(edge *domain.Edge) error {
	_, err := t.tx.Exec(`
		INSERT OR REPLACE INTO edges (source_path, target_jd_id, link_text)
		VALUES (?, ?, ?)
	`, edge.SourcePath, edge.TargetJDID, edge.LinkText)
	return err
}

// UpdateEdgeTarget updates all edges pointing to oldTargetJDID
func (t *indexTx) UpdateEdgeTarget(oldTargetJDID, newTargetJDID, newLinkText string) error {
	_, err := t.tx.Exec(`
		UPDATE edges
		SET target_jd_id = ?, link_text = ?
		WHERE target_jd_id = ?
	`, newTargetJDID, newLinkText, oldTargetJDID)
	return err
}

// Commit commits the transaction
func (t *indexTx) Commit() error {
	return t.tx.Commit()
}

// Rollback aborts the transaction
func (t *indexTx) Rollback() error {
	return t.tx.Rollback()
}
