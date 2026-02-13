# Libraio

A toolkit for managing Johnny Decimal vaults in Obsidian.

## Binaries

| Binary | Description |
|--------|-------------|
| `libraio` | TUI browser and manager |
| `libraio-cli` | Command-line interface |
| `libraio-mcp` | MCP server for AI tool integration |

## Install

```bash
make install        # TUI
make install-cli    # CLI
make install-mcp    # MCP server
```

## Usage

```bash
libraio                          # Launch TUI
libraio-cli list scopes          # CLI: list scopes
libraio-mcp                      # Start MCP server on stdio
libraio-mcp --vault ~/my-vault   # Custom vault path
```

### TUI Keys

| Key | Action |
|-----|--------|
| `j/k` | Navigate |
| `space` | Toggle / Open |
| `y` | Copy ID |
| `n` | New item |
| `a` | Archive |
| `m` | Move item |
| `/` | Search |
| `?` | Help |
| `q` | Quit |

### MCP Tools

**Read**: `list`, `search`, `tree`, `read_jdex`, `resolve_path`
**Write**: `create`, `move`, `rename`, `archive`, `unarchive`, `delete`

### Configuration

All binaries use the same vault path resolution:
1. `LIBRAIO_VAULT` environment variable
2. `--vault` flag (CLI and MCP)
3. Default: `~/Documents/bag_of_holding`

## License

MIT
