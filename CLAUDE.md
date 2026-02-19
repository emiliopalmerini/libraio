# Claude Code Instructions

## Project Overview

Libraio manages Obsidian vaults organized with the Johnny Decimal system. Two binaries: `libraio` (TUI), `libraio-cli` (CLI).

Hexagonal architecture: domain defines ports, adapters implement them. Organized by domain, not technical layer.

## Johnny Decimal ID Format

**IMPORTANT**: All IDs must include the scope prefix.

- **Scope**: `S01`, `S02`, `S03`
- **Area**: `S01.10-19` (range format)
- **Category**: `S01.11`
- **Item**: `S01.11.15`

## Domain Conventions (not in code)

### Standard Zeros

Reserved ID slots at every level:

| Slot | Purpose |
|------|---------|
| `.01` | Inbox |
| `.02` | Tasks |
| `.03` | Templates |
| `.04` | Links |
| `.08` | Someday |
| `.09` | Archive |

- Regular content IDs start at `.11` (`.10` is skipped as buffer)
- Standard zeros use a "for" suffix: `S01.11.01 Inbox for S01.11`
- Area management uses the `.X0` category: `S01.10 Management for S01.10-19`

### Archive Behavior

- **Item → Category archive**: renamed to `[Archived] Description`, loses its ID, links updated
- **Category → Area archive**: preserves ID and all items
- **Area → Scope archive**: preserves full structure

### JDex Entries

JDex entries live in the **BagOfHoldingIndexDB** Notion database — not as files in the vault. The vault only holds actual content. Existing JDex files in the vault are read-only (used for display in the TUI but no longer created or updated).

### File Naming

- Use Title Case for descriptions
- Avoid special characters except hyphens and spaces

## Configuration

Both binaries share vault path config via `internal/config`:
1. `LIBRAIO_VAULT` env var (highest priority)
2. `--vault` flag (CLI only)
3. Default: `~/Documents/bag_of_holding`
