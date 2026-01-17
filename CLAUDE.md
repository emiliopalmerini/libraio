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
├── S01.00-09 Management for S01/          [Scope-level Management]
│   ├── S01.01 Inbox for S01.00-09/
│   ├── S01.02 Tasks for S01.00-09/
│   ├── S01.03 Templates for S01.00-09/
│   └── S01.09 Archive for S01.00-09/
├── S01.10-19 Lifestyle/                   [Area]
│   ├── S01.10 Management for S01.10-19/   [Area Management Category]
│   │   ├── S01.10.01 Inbox for S01.10-19/
│   │   └── S01.10.09 Archive for S01.10-19/
│   └── S01.11 Entertainment/              [Category]
│       ├── S01.11.01 Inbox for S01.11/    [Category Standard Zero]
│       ├── S01.11.09 Archive for S01.11/  [Category Archive]
│       └── S01.11.11 Theatre, 2025 Season/ [Item]
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

This project follows the [Standard Zeros](https://johnnydecimal.com/10-19-concepts/12-advanced/12.03-the-standard-zeros/) convention at two levels:

1. **Scope-level** - 00-09 Management area for scope-wide items
2. **Category-level** - .00-.09 IDs within each category

### Reserved ID Slots

| Slot | Purpose | Area Example | Category Example |
|------|---------|--------------|------------------|
| `.01` | Inbox | `S01.10.01 Inbox for S01.10-19` | `S01.11.01 Inbox for S01.11` |
| `.02` | Tasks | `S01.10.02 Tasks for S01.10-19` | `S01.11.02 Tasks for S01.11` |
| `.03` | Templates | `S01.10.03 Templates for S01.10-19` | `S01.11.03 Templates for S01.11` |
| `.04` | Links | `S01.10.04 Links for S01.10-19` | `S01.11.04 Links for S01.11` |
| `.08` | Someday | `S01.10.08 Someday for S01.10-19` | `S01.11.08 Someday for S01.11` |
| `.09` | Archive | `S01.10.09 Archive for S01.10-19` | `S01.11.09 Archive for S01.11` |

### Regular Content

Regular content IDs start at `.11` (`.10` is intentionally skipped as a buffer).

### Archive System

Archives exist at three levels with hierarchical archiving:
- **Category-level**: `S01.11.09 Archive for S01.11` - for items within that category
- **Area-level**: `S01.10.09 Archive for S01.10-19` - for categories within that area
- **Scope-level**: `S01.09 Archive for S01.00-09` - for areas within that scope

Archive behavior:
- **Item → Category archive**: Item is renamed to `[Archived] Description` and moves to the category's `.09` folder
  - `S01.11.15 Theatre` → `S01.11.09 Archive for S01.11/[Archived] Theatre`
  - Obsidian links are updated: `[[S01.11.15 Theatre]]` → `[[[Archived] Theatre]]`
- **Category → Area archive**: Category moves to the area's `.X0.09` folder, preserving its ID and all items
  - `S01.11 Entertainment` → `S01.10.09 Archive for S01.10-19/S01.11 Entertainment`
- **Area → Scope archive**: Area moves to the scope's `.09` folder, preserving its structure

## JDex (Johnny Decimal Index)

The JDex is the master record of every ID in the system. Each item folder contains a JDex file **named after the folder** (e.g., `S01.11.01 Inbox for S01.11/S01.11.01 Inbox for S01.11.md`).

### JDex File Naming

JDex files match their parent folder name:
- `S01.11.01 Inbox for S01.11/S01.11.01 Inbox for S01.11.md`
- `S01.10.01 Inbox for S01.10-19/S01.10.01 Inbox for S01.10-19.md`
- `S01.21.11 CSharp/S01.21.11 CSharp.md`

### Standard JDex Entry Format

Every ID folder should contain a JDex file with this structure:

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

- **Every ID folder** (Scope.XX.YY format) must contain a JDex file named after the folder
- The JDex file serves as the main entry point and overview for that ID's content
- Legacy `README.md` files are still supported for backwards compatibility

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
5. Create JDex file named after the folder with frontmatter

### Standard Zero Naming Convention

All standard zeros include a "for" suffix indicating their scope:

**Area management categories (.X0)**:
- `S01.10.01 Inbox for S01.10-19`
- `S01.10.02 Tasks for S01.10-19`

**Regular categories**:
- `S01.11.01 Inbox for S01.11`
- `S01.11.02 Tasks for S01.11`

## Code Conventions

- Tree nodes are lazily loaded via `LoadChildren()`
- Items have JDex files (named after folder) in their directories
- Legacy README.md files are supported for backwards compatibility
- Search uses fuzzy matching with scoring
- Archived items are prefixed with `[Archived]` and lose their Johnny Decimal ID
- Categories and areas preserve their IDs when archived (they are moved, not renamed)
