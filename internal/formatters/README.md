# Task Output Formatters

This package will provide multiple output formats for task data.

## Current Status
Currently, the CLI uses two output modes:
- **Table** (default): Pretty-printed tables using pterm
- **JSON**: Machine-readable JSON output via `--json` flag

## Future Implementation

See [`docs/future-enhancements/output-formats.md`](../../docs/future-enhancements/output-formats.md) for the complete implementation plan.

### Planned Formatters

1. **MarkdownFormatter** - Export tasks as markdown documents
2. **YAMLFormatter** - YAML format for config-style exports
3. **CSVFormatter** - CSV for spreadsheet imports
4. **TableFormatter** - Refactored from existing table output
5. **JSONFormatter** - Refactored from existing JSON output

### Implementation Steps

1. Create individual formatter files:
   - `json_formatter.go`
   - `table_formatter.go`
   - `markdown_formatter.go`
   - `yaml_formatter.go`
   - `csv_formatter.go`

2. Update CLI to use `--format` flag instead of `--json`

3. Add format-specific dependencies:
   - `gopkg.in/yaml.v3` for YAML
   - Standard library `encoding/csv` for CSV

### Usage Example (Future)
```bash
shark task get T-E04-F01-001 --format=markdown > task.md
shark task list --format=yaml > tasks.yaml
shark task list --format=csv > tasks.csv
```

## Architecture

```
formatters/
├── formatter.go          # Interface definition (exists)
├── json_formatter.go     # JSON implementation (TODO)
├── table_formatter.go    # Table implementation (TODO)
├── markdown_formatter.go # Markdown implementation (TODO)
├── yaml_formatter.go     # YAML implementation (TODO)
└── csv_formatter.go      # CSV implementation (TODO)
```
