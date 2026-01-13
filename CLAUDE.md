# Claude Code Instructions

## Project Overview

Librarian is a Go TUI application for managing Obsidian vaults organized with the Johnny Decimal system. It uses Bubble Tea for the terminal interface.

## Architecture

Hexagonal architecture with DDD:

```
cmd/librarian/main.go    # Entry point
internal/
  domain/                # Business logic, models, ID parsing
  ports/                 # Interfaces (VaultRepository, EditorOpener)
  adapters/
    filesystem/          # VaultRepository implementation
    editor/              # Editor integration
    tui/                 # Bubble Tea UI
      app.go             # Main app orchestrator
      views/             # Browser, Create, Help views
      styles/            # Theming
```

## Johnny Decimal ID Format

- Scope: `S00`, `S01`, `S02`, `S03`
- Area: `S01.10-19` (range within scope)
- Category: `S01.11` (single category)
- Item: `S01.11.15` (individual item)

## Key Files

- `internal/domain/jdecimal.go` - ID parsing and validation
- `internal/domain/vault.go` - Domain models (Scope, Area, Category, Item, TreeNode)
- `internal/adapters/filesystem/repository.go` - Filesystem operations
- `internal/adapters/tui/views/browser.go` - Main tree browser with search

## Build & Run

```bash
make build    # Build binary
make run      # Build and run
make install  # Install to ~/.local/bin
```

## Conventions

- Tree nodes are lazily loaded via `LoadChildren()`
- Items have README.md files in their directories
- Archive categories end in `.X9` (e.g., `S01.19`)
- Search uses fuzzy matching with scoring
