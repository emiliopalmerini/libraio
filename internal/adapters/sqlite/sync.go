package sqlite

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"libraio/internal/domain"
)

// Link pattern for Obsidian wiki links: [[S01.XX.YY...]]
var linkPattern = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// JD ID pattern: S0X.XX.XX or S0X.XX
var jdIDPattern = regexp.MustCompile(`^(S0[0-9]\.[0-9][0-9](?:\.[0-9][0-9])?)`)

// SyncFull performs a complete rebuild of the index
func (idx *Index) SyncFull() (*domain.SyncStats, error) {
	start := time.Now()
	stats := &domain.SyncStats{}

	// Clear existing data
	if _, err := idx.db.Exec(`DELETE FROM nodes`); err != nil {
		return nil, err
	}
	if _, err := idx.db.Exec(`DELETE FROM edges`); err != nil {
		return nil, err
	}

	// Walk the vault
	err := filepath.Walk(idx.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		relPath, _ := filepath.Rel(idx.vaultPath, path)
		stats.FilesScanned++

		if info.IsDir() {
			// Check if directory is a JD node
			jdID, jdType := extractJDInfo(info.Name())
			if jdType != domain.IDTypeUnknown {
				node := &domain.IndexNode{
					Path:   relPath,
					JDID:   jdID,
					JDType: jdType,
					Name:   extractDescription(info.Name()),
					Mtime:  info.ModTime().Unix(),
				}
				if err := idx.insertNode(node); err != nil {
					return nil // Continue on error
				}
				stats.NodesAdded++
			}
		} else if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			// Index markdown file for links
			node := &domain.IndexNode{
				Path:   relPath,
				Name:   info.Name(),
				JDType: domain.IDTypeUnknown,
				Mtime:  info.ModTime().Unix(),
			}
			if err := idx.insertNode(node); err != nil {
				return nil // Continue on error
			}
			stats.NodesAdded++

			// Parse and index links
			edges, err := parseLinksInFile(filepath.Join(idx.vaultPath, relPath), relPath)
			if err == nil {
				for _, edge := range edges {
					if err := idx.insertEdge(&edge); err == nil {
						stats.EdgesAdded++
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return stats, err
	}

	// Update last sync time
	idx.db.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES ('last_sync_time', ?)`,
		time.Now().Unix())

	stats.Duration = time.Since(start)
	return stats, nil
}

// SyncIncremental updates only files that changed since last sync
func (idx *Index) SyncIncremental() (*domain.SyncStats, error) {
	start := time.Now()
	stats := &domain.SyncStats{}

	// Get last sync time
	var lastSyncUnix int64
	idx.db.QueryRow(`SELECT value FROM meta WHERE key = 'last_sync_time'`).Scan(&lastSyncUnix)

	// Track existing paths to detect deletions
	existingPaths := make(map[string]bool)
	rows, err := idx.db.Query(`SELECT path FROM nodes`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var path string
		rows.Scan(&path)
		existingPaths[path] = true
	}
	rows.Close()

	// Track paths we've seen during this walk
	seenPaths := make(map[string]bool)

	// Walk the vault
	err = filepath.Walk(idx.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		relPath, _ := filepath.Rel(idx.vaultPath, path)
		seenPaths[relPath] = true
		stats.FilesScanned++

		// Check if file is new or modified
		mtime := info.ModTime().Unix()
		needsUpdate := mtime > lastSyncUnix || !existingPaths[relPath]

		if !needsUpdate {
			return nil
		}

		if info.IsDir() {
			jdID, jdType := extractJDInfo(info.Name())
			if jdType != domain.IDTypeUnknown {
				node := &domain.IndexNode{
					Path:   relPath,
					JDID:   jdID,
					JDType: jdType,
					Name:   extractDescription(info.Name()),
					Mtime:  mtime,
				}
				if existingPaths[relPath] {
					idx.updateNode(node)
					stats.NodesUpdated++
				} else {
					idx.insertNode(node)
					stats.NodesAdded++
				}
			}
		} else if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			node := &domain.IndexNode{
				Path:   relPath,
				Name:   info.Name(),
				JDType: domain.IDTypeUnknown,
				Mtime:  mtime,
			}

			if existingPaths[relPath] {
				idx.updateNode(node)
				stats.NodesUpdated++
				// Delete old edges
				idx.db.Exec(`DELETE FROM edges WHERE source_path = ?`, relPath)
			} else {
				idx.insertNode(node)
				stats.NodesAdded++
			}

			// Parse and index links
			edges, err := parseLinksInFile(filepath.Join(idx.vaultPath, relPath), relPath)
			if err == nil {
				for _, edge := range edges {
					if err := idx.insertEdge(&edge); err == nil {
						stats.EdgesAdded++
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return stats, err
	}

	// Delete nodes that no longer exist
	for path := range existingPaths {
		if !seenPaths[path] {
			idx.db.Exec(`DELETE FROM nodes WHERE path = ?`, path)
			idx.db.Exec(`DELETE FROM edges WHERE source_path = ?`, path)
			stats.NodesDeleted++
		}
	}

	// Update last sync time
	idx.db.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES ('last_sync_time', ?)`,
		time.Now().Unix())

	stats.Duration = time.Since(start)
	return stats, nil
}

// insertNode inserts a node into the database
func (idx *Index) insertNode(node *domain.IndexNode) error {
	_, err := idx.db.Exec(`
		INSERT INTO nodes (path, jd_id, jd_type, name, mtime)
		VALUES (?, ?, ?, ?, ?)
	`, node.Path, nullString(node.JDID), nullString(node.JDType.String()), node.Name, node.Mtime)
	return err
}

// updateNode updates an existing node
func (idx *Index) updateNode(node *domain.IndexNode) error {
	_, err := idx.db.Exec(`
		UPDATE nodes SET jd_id = ?, jd_type = ?, name = ?, mtime = ?
		WHERE path = ?
	`, nullString(node.JDID), nullString(node.JDType.String()), node.Name, node.Mtime, node.Path)
	return err
}

// insertEdge inserts an edge into the database
func (idx *Index) insertEdge(edge *domain.Edge) error {
	_, err := idx.db.Exec(`
		INSERT OR REPLACE INTO edges (source_path, target_jd_id, link_text)
		VALUES (?, ?, ?)
	`, edge.SourcePath, edge.TargetJDID, edge.LinkText)
	return err
}

// parseLinksInFile extracts all JD links from a markdown file
func parseLinksInFile(fullPath, relPath string) ([]domain.Edge, error) {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	var edges []domain.Edge
	matches := linkPattern.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		linkContent := match[1]
		jdMatch := jdIDPattern.FindString(linkContent)
		if jdMatch != "" {
			edges = append(edges, domain.Edge{
				SourcePath: relPath,
				TargetJDID: jdMatch,
				LinkText:   match[0],
			})
		}
	}

	return edges, nil
}

// extractJDInfo extracts the JD ID and type from a folder name
func extractJDInfo(name string) (string, domain.IDType) {
	// Try to parse as JD ID
	jdID := domain.ExtractID(name)
	if jdID == "" {
		return "", domain.IDTypeUnknown
	}
	return jdID, domain.ParseIDType(jdID)
}

// extractDescription extracts the description from a JD folder name
func extractDescription(name string) string {
	// Format: "S01.11.15 Theatre" -> "Theatre"
	parts := strings.SplitN(name, " ", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return name
}

// nullString returns nil for empty strings (for nullable columns)
func nullString(s string) interface{} {
	if s == "" || s == "unknown" {
		return nil
	}
	return s
}
