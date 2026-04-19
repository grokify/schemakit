# SchemaKit

JSON Schema toolkit for Go developers.

## Overview

**schemakit** is a toolkit for working with JSON Schema in Go projects. It provides:

- **Linting** - Validate schemas for compatibility with statically-typed languages
- **Generation** - Generate JSON Schema from Go struct types
- **Documentation** - Generate Markdown specification docs from Go types

## Quick Start

```bash
# Install
go install github.com/grokify/schemakit/cmd/schemakit@latest

# Generate JSON Schema from Go types
schemakit generate github.com/myorg/myproject/types Config -o schema.json

# Generate Markdown documentation
schemakit doc github.com/myorg/myproject/types Config -o spec.md

# Lint a schema
schemakit lint schema.json
```

## Commands

| Command | Description |
|---------|-------------|
| `schemakit lint` | Check schemas for static type compatibility |
| `schemakit generate` | Generate JSON Schema from Go struct types |
| `schemakit doc` | Generate Markdown documentation from Go types |
| `schemakit version` | Print version information |

## Go-First Workflow

schemakit supports a Go-first approach where Go structs are the source of truth:

```
Go Structs (with doc comments)
         │
         ├──► schemakit generate ──► JSON Schema
         │
         └──► schemakit doc ──► Markdown Specification
```

This ensures your JSON Schema and documentation are always in sync with your Go types.

## Use Cases

- **API Development** - Generate schemas and docs from Go request/response types
- **Configuration** - Document config structs with required/optional fields
- **Specifications** - Create human-readable specs from Go type definitions
- **Validation** - Lint schemas before code generation to catch issues early
