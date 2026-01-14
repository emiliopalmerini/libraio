# Claude Code Instructions

## Project Overview

Libraio is a Go TUI application for managing Obsidian vaults organized with the Johnny Decimal system. It uses Bubble Tea for the terminal interface.

## Architecture

Hexagonal architecture with DDD:

```
cmd/libraio/main.go      # Entry point
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

**IMPORTANT**: All areas, categories and IDs must include the scope prefix.

- **Scope**: S + Number (S01, S02, S03)
- **Area**: Scope.XX-YY (e.g., `S01.10-19 Lifestyle`)
- **Category**: Scope.XX (e.g., `S01.11 Entertainment`)
- **Item**: Scope.XX.YY (e.g., `S01.11.15 Season 7 Episode 1`)

### Example Structure

```
S01 Me/                                    [Scope]
├── S01.10-19 Lifestyle/                   [Area]
│   ├── S01.11 Entertainment/              [Category]
│   │   └── S01.11.11 Theatre, 2025 Season/ [Item]
│   ├── S01.12 Recipes/                    [Category]
│   │   ├── S01.12.11 Italian/             [Item]
│   │   └── S01.12.12 African/             [Item]
│   └── S01.19 Archive/                    [Archive]
```

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

## Standard Zeros Convention

This project follows the [Standard Zeros](https://johnnydecimal.com/10-19-concepts/12-advanced/12.03-the-standard-zeros/) convention. Management items are distributed at multiple levels rather than a separate S00 scope.

### Reserved ID Slots (.00-.09)

Within each category, IDs `.00` through `.09` are reserved for management purposes:

| ID | Purpose | Example |
|----|---------|---------|
| `.00` | JDex (index data) | `S01.21.00 JDex` |
| `.01` | Inbox (items to sort) | `S01.21.01 Inbox` |
| `.02` | Tasks & projects | `S01.21.02 Tasks` |
| `.03` | Templates | `S01.21.03 Templates` |
| `.04` | Links & references | `S01.21.04 Links` |
| `.08` | Someday (future items) | `S01.21.08 Someday` |
| `.09` | Archive (unorganized) | `S01.21.09 Archive` |

### Hierarchy Preference

Prefer the most specific zero available:
1. **Category level** (`.0X` IDs) - most preferred
2. **Area level** (`.X0` categories) - for area-wide items
3. **Scope level** - only for items spanning multiple areas

### Regular Content

Regular content IDs start at `.11` (`.10` is intentionally skipped as a buffer).

### Archive Categories

Each area has a dedicated archive category using the `.X9` pattern (e.g., `S01.19 Archive`).

## JDex (Johnny Decimal Index)

The JDex is the master record of every ID in the system. Each `README.md` file in an ID folder serves as a JDex entry.

### Standard JDex Entry Format

Every ID folder should contain a `README.md` with this structure:

```yaml
---
aliases:
  - S01.21.11 CSharp          # Full ID for searchability
location: Obsidian            # Where content lives
tags:
  - jdex
  - index
---

# S01.21.11 CSharp

Brief description of what this ID contains.
```

### Index File Rules

- **Every ID folder** (Scope.XX.YY format) should contain a `README.md` as its JDex entry
- The README.md serves as the main entry point and overview for that ID's content

## File Naming Rules

1. **ALWAYS** use full Johnny Decimal ID with scope prefix in folder names
   - Correct: `S01.11 Entertainment`
   - Wrong: `11 Entertainment`
2. Keep descriptions concise and clear
3. Use Title Case for descriptions
4. Avoid special characters except hyphens and spaces

## Creating New Items

1. Determine which scope (S01, S02, S03) it belongs to
2. Find the appropriate area and category
3. Assign the next available ID:
   - For system items (inbox, templates, etc.), use Standard Zeros (`.01`-`.09`)
   - For regular content, use IDs starting at `.11`
4. Create folder using format: `Scope.XX.YY Description`
5. Create README.md with JDex frontmatter

## Code Conventions

- Tree nodes are lazily loaded via `LoadChildren()`
- Items have README.md files in their directories
- Search uses fuzzy matching with scoring
