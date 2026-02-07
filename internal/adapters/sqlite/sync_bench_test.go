package sqlite

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkSyncFull benchmarks just the sync operation (DB already open)
func BenchmarkSyncFull(b *testing.B) {
	vaultPath := os.Getenv("VAULT_PATH")
	if vaultPath == "" {
		b.Skip("VAULT_PATH not set")
	}

	idx := NewIndex()
	if err := idx.Open(vaultPath); err != nil {
		b.Fatalf("failed to open index: %v", err)
	}
	defer func() {
		if err := idx.Close(); err != nil {
			b.Fatalf("failed to close index: %v", err)
		}
	}()

	b.ResetTimer()
	for b.Loop() {
		_, err := idx.SyncFull()
		if err != nil {
			b.Fatalf("sync failed: %v", err)
		}
	}
}

// BenchmarkFullStartup benchmarks cold startup: open + full sync + close (no existing DB)
func BenchmarkFullStartup(b *testing.B) {
	vaultPath := os.Getenv("VAULT_PATH")
	if vaultPath == "" {
		b.Skip("VAULT_PATH not set")
	}

	// Use a temp DB path for each run
	tmpDir := b.TempDir()

	b.Setenv("XDG_DATA_HOME", tmpDir)

	b.ResetTimer()
	for b.Loop() {
		idx := NewIndex()
		if err := idx.Open(vaultPath); err != nil {
			b.Fatalf("failed to open index: %v", err)
		}

		_, err := idx.SyncFull()
		if err != nil {
			b.Fatalf("sync failed: %v", err)
		}

		if err := idx.Close(); err != nil {
			b.Fatalf("failed to close index: %v", err)
		}

		// Clean up for next iteration
		if err := os.RemoveAll(filepath.Join(tmpDir, "libraio")); err != nil {
			b.Fatalf("failed to clean up: %v", err)
		}
	}
}

// BenchmarkWarmStartup benchmarks warm startup: open + incremental sync (DB exists, no changes)
func BenchmarkWarmStartup(b *testing.B) {
	vaultPath := os.Getenv("VAULT_PATH")
	if vaultPath == "" {
		b.Skip("VAULT_PATH not set")
	}

	tmpDir := b.TempDir()
	b.Setenv("XDG_DATA_HOME", tmpDir)

	// First, create the DB with a full sync
	idx := NewIndex()
	if err := idx.Open(vaultPath); err != nil {
		b.Fatalf("failed to open index: %v", err)
	}
	if _, err := idx.SyncFull(); err != nil {
		b.Fatalf("initial sync failed: %v", err)
	}
	if err := idx.Close(); err != nil {
		b.Fatalf("failed to close index: %v", err)
	}

	// Wait a moment to ensure mtime won't trigger updates
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for b.Loop() {
		idx := NewIndex()
		if err := idx.Open(vaultPath); err != nil {
			b.Fatalf("failed to open index: %v", err)
		}

		_, err := idx.SyncIncremental()
		if err != nil {
			b.Fatalf("sync failed: %v", err)
		}

		if err := idx.Close(); err != nil {
			b.Fatalf("failed to close index: %v", err)
		}
	}
}
