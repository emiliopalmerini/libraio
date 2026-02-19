# Libraio

A toolkit for managing Johnny Decimal vaults in Obsidian.

## Binaries

| Binary | Description |
|--------|-------------|
| `libraio` | TUI browser and manager |
| `libraio-cli` | Command-line interface |

## Install

```bash
make install        # TUI
make install-cli    # CLI
```

## Usage

```bash
libraio                          # Launch TUI
libraio-cli list scopes          # CLI: list scopes
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

### Configuration

All binaries use the same vault path resolution:
1. `LIBRAIO_VAULT` environment variable
2. `--vault` flag (CLI)
3. Default: `~/Documents/bag_of_holding`

## License

MIT
