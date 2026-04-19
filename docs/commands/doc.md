# schemakit doc

Generate Markdown specification documentation from Go struct types.

## Usage

```bash
schemakit doc <package> <type> [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output file (default: stdout) |
| `-t, --title` | Specification title (default: derived from type name) |
| `-v, --version` | Specification version (e.g., v0.4.0) |
| `--prepend` | Prepend content from this file (for custom headers/examples) |

## Examples

```bash
# Generate to stdout
schemakit doc github.com/grokify/threat-model-spec/ir ThreatModel

# With title and version
schemakit doc -t "Threat Model Specification" -v v0.4.0 \
  github.com/grokify/threat-model-spec/ir ThreatModel -o spec.md

# With custom header prepended
schemakit doc --prepend header.md \
  github.com/grokify/threat-model-spec/ir ThreatModel -o spec.md
```

## What Gets Extracted

### From Struct Types

- Type names and descriptions (from doc comments)
- Field names (Go and JSON)
- Field types
- Required vs optional (based on `omitempty`)
- Field descriptions (from doc comments)

### From Enum Types

- Enum type names (`type X string`)
- Enum values (from `const` blocks)
- Value descriptions (from const doc comments)

## Output Structure

The generated Markdown includes:

1. **Title** - With optional version
2. **Package Description** - From package doc comment
3. **Types Table of Contents** - Links to all struct types
4. **Enums Table of Contents** - Links to all enum types
5. **Type Reference** - Each type with Required/Optional field tables
6. **Enum Reference** - Each enum with value tables

## Example Output

```markdown
# MySpec Specification v1.0.0

Package myspec provides types for...

## Types

- [Config](#config)
- [Database](#database)

## Enums

- [Environment](#environment)

---

# Type Reference

## Config

Configuration for the application.

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Application name |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `debug` | bool | Enable debug mode |

---

# Enum Reference

## Environment

Deployment environment.

| Value | Description |
|-------|-------------|
| `dev` | Development environment |
| `prod` | Production environment |
```

## See Also

- [Spec Documentation Guide](../guides/spec-documentation.md) - Creating human-audience specs
- [Go-First Workflow](../guides/go-first-workflow.md) - Complete workflow guide
