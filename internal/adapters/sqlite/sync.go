package sqlite

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"libraio/internal/domain"
)

// mdFile holds info about a markdown file to process
type mdFile struct {
	fullPath string
	relPath  string
	mtime    int64
}

// Link pattern for Obsidian wiki links: [[S01.XX.YY...]]
var linkPattern = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// JD ID pattern: S0X.XX.XX or S0X.XX
var jdIDPattern = regexp.MustCompile(`^(S0[0-9]\.[0-9][0-9](?:\.[0-9][0-9])?)`)

// SyncFull performs a complete rebuild of the index
func (idx *Index) SyncFull() (*domain.SyncStats, error) {
	start := time.Now()
	stats := &domain.SyncStats{}

	// Begin transaction - CRITICAL for performance
	tx, err := idx.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Clear existing data
	if _, err := tx.Exec(`DELETE FROM nodes`); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM edges`); err != nil {
		return nil, err
	}

	// Prepare statements once
	insertNodeStmt, err := tx.Prepare(`
		INSERT INTO nodes (path, jd_id, jd_type, name, mtime)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer insertNodeStmt.Close()

	insertEdgeStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO edges (source_path, target_jd_id, link_text)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer insertEdgeStmt.Close()

	// Collect markdown files for parallel processing
	// Pre-allocate with estimated capacity
	mdFiles := make([]mdFile, 0, 1024)

	// Use WalkDir - faster than Walk (avoids redundant stat calls)
	err = filepath.WalkDir(idx.vaultPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		name := d.Name()

		// Skip hidden directories
		if d.IsDir() && len(name) > 0 && name[0] == '.' {
			return filepath.SkipDir
		}

		relPath, _ := filepath.Rel(idx.vaultPath, path)
		stats.FilesScanned++

		if d.IsDir() {
			jdID, jdType := extractJDInfo(name)
			if jdType != domain.IDTypeUnknown {
				info, err := d.Info()
				if err != nil {
					return nil
				}
				_, err = insertNodeStmt.Exec(
					relPath,
					nullString(jdID),
					nullString(jdType.String()),
					extractDescription(name),
					info.ModTime().Unix(),
				)
				if err == nil {
					stats.NodesAdded++
				}
			}
		} else if len(name) > 3 && strings.EqualFold(name[len(name)-3:], ".md") {
			// Collect for parallel processing
			info, err := d.Info()
			if err != nil {
				return nil
			}
			mdFiles = append(mdFiles, mdFile{
				fullPath: path,
				relPath:  relPath,
				mtime:    info.ModTime().Unix(),
			})
		}

		return nil
	})

	if err != nil {
		return stats, err
	}

	// Process markdown files in parallel
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8 // Cap workers to avoid too many open files
	}

	// Channel for files to process
	fileCh := make(chan mdFile, len(mdFiles))
	for _, f := range mdFiles {
		fileCh <- f
	}
	close(fileCh)

	// Channel for results
	type fileResult struct {
		relPath string
		name    string
		mtime   int64
		edges   []domain.Edge
	}
	resultCh := make(chan fileResult, len(mdFiles))

	// Spawn workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range fileCh {
				edges, _ := parseLinksInFile(f.fullPath, f.relPath)
				resultCh <- fileResult{
					relPath: f.relPath,
					name:    filepath.Base(f.fullPath),
					mtime:   f.mtime,
					edges:   edges,
				}
			}
		}()
	}

	// Close results channel when workers done
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Insert nodes and edges from results
	for r := range resultCh {
		_, err := insertNodeStmt.Exec(r.relPath, nil, nil, r.name, r.mtime)
		if err == nil {
			stats.NodesAdded++
		}
		for _, edge := range r.edges {
			_, err := insertEdgeStmt.Exec(edge.SourcePath, edge.TargetJDID, edge.LinkText)
			if err == nil {
				stats.EdgesAdded++
			}
		}
	}

	// Update last sync time
	tx.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES ('last_sync_time', ?)`,
		time.Now().Unix())

	if err := tx.Commit(); err != nil {
		return stats, err
	}

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

	// Begin transaction - CRITICAL for performance
	tx, err := idx.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Prepare statements once
	insertNodeStmt, err := tx.Prepare(`
		INSERT INTO nodes (path, jd_id, jd_type, name, mtime)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer insertNodeStmt.Close()

	updateNodeStmt, err := tx.Prepare(`
		UPDATE nodes SET jd_id = ?, jd_type = ?, name = ?, mtime = ?
		WHERE path = ?
	`)
	if err != nil {
		return nil, err
	}
	defer updateNodeStmt.Close()

	insertEdgeStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO edges (source_path, target_jd_id, link_text)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer insertEdgeStmt.Close()

	deleteEdgesStmt, err := tx.Prepare(`DELETE FROM edges WHERE source_path = ?`)
	if err != nil {
		return nil, err
	}
	defer deleteEdgesStmt.Close()

	deleteNodeStmt, err := tx.Prepare(`DELETE FROM nodes WHERE path = ?`)
	if err != nil {
		return nil, err
	}
	defer deleteNodeStmt.Close()

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
				if existingPaths[relPath] {
					updateNodeStmt.Exec(
						nullString(jdID),
						nullString(jdType.String()),
						extractDescription(info.Name()),
						mtime,
						relPath,
					)
					stats.NodesUpdated++
				} else {
					insertNodeStmt.Exec(
						relPath,
						nullString(jdID),
						nullString(jdType.String()),
						extractDescription(info.Name()),
						mtime,
					)
					stats.NodesAdded++
				}
			}
		} else if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			if existingPaths[relPath] {
				updateNodeStmt.Exec(nil, nil, info.Name(), mtime, relPath)
				stats.NodesUpdated++
				// Delete old edges
				deleteEdgesStmt.Exec(relPath)
			} else {
				insertNodeStmt.Exec(relPath, nil, nil, info.Name(), mtime)
				stats.NodesAdded++
			}

			// Parse and index links
			edges, err := parseLinksInFile(filepath.Join(idx.vaultPath, relPath), relPath)
			if err == nil {
				for _, edge := range edges {
					_, err := insertEdgeStmt.Exec(edge.SourcePath, edge.TargetJDID, edge.LinkText)
					if err == nil {
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
			deleteNodeStmt.Exec(path)
			deleteEdgesStmt.Exec(path)
			stats.NodesDeleted++
		}
	}

	// Update last sync time
	tx.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES ('last_sync_time', ?)`,
		time.Now().Unix())

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return stats, err
	}

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
