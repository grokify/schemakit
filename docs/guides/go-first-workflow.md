# Go-First Workflow

This guide explains the Go-first approach to JSON Schema and documentation, where Go structs are the single source of truth.

## Philosophy

Traditional schema-first development requires maintaining schemas and code in sync. The Go-first approach inverts this:

```
                    ┌─────────────────┐
                    │   Go Structs    │
                    │ (Source of      │
                    │     Truth)      │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
        ┌───────────┐  ┌───────────┐  ┌───────────┐
        │   JSON    │  │  Markdown │  │    Go     │
        │  Schema   │  │   Spec    │  │ Validation│
        └───────────┘  └───────────┘  └───────────┘
```

## Benefits

- **Single source of truth** - No schema/code drift
- **Type safety** - Go compiler catches errors
- **Doc comments** - Documentation lives with code
- **IDE support** - Full autocomplete and refactoring

## Complete Workflow

### 1. Define Types in Go

```go
// Package config defines application configuration types.
package config

// Config is the root configuration type.
type Config struct {
    // App contains application settings.
    App AppConfig `json:"app"`

    // Database contains database connection settings.
    Database DatabaseConfig `json:"database,omitempty"`
}

// AppConfig contains application-level settings.
type AppConfig struct {
    // Name is the application name.
    Name string `json:"name"`

    // Environment is the deployment environment.
    Environment Environment `json:"environment"`

    // Debug enables debug mode.
    Debug bool `json:"debug,omitempty"`
}

// Environment represents deployment environments.
type Environment string

const (
    // EnvironmentDev is the development environment.
    EnvironmentDev Environment = "dev"

    // EnvironmentStaging is the staging environment.
    EnvironmentStaging Environment = "staging"

    // EnvironmentProd is the production environment.
    EnvironmentProd Environment = "prod"
)
```

### 2. Generate JSON Schema

```bash
schemakit generate github.com/myorg/myproject/config Config \
  -o schema/config.schema.json
```

### 3. Generate Documentation

```bash
schemakit doc github.com/myorg/myproject/config Config \
  -t "Configuration Specification" -v v1.0.0 \
  -o docs/config-spec.md
```

### 4. Validate Schema (Optional)

```bash
schemakit lint schema/config.schema.json
```

## Project Structure

```
myproject/
├── config/
│   ├── config.go         # Go types (source of truth)
│   └── config_test.go    # Tests
├── schema/
│   └── config.schema.json  # Generated
├── docs/
│   ├── header.md           # Human-authored
│   └── config-spec.md      # Generated (with header)
└── Makefile
```

## Makefile Integration

```makefile
VERSION ?= v1.0.0

.PHONY: schema docs lint

schema:
	schemakit generate github.com/myorg/myproject/config Config \
	  -o schema/config.schema.json

docs:
	schemakit doc --prepend docs/header.md \
	  -t "Configuration Specification" -v $(VERSION) \
	  github.com/myorg/myproject/config Config \
	  -o docs/config-spec.md

lint:
	schemakit lint schema/config.schema.json

all: schema docs lint
```

## CI Integration

```yaml
# .github/workflows/schema.yml
name: Schema
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install schemakit
        run: go install github.com/grokify/schemakit/cmd/schemakit@latest

      - name: Generate schema
        run: make schema

      - name: Lint schema
        run: make lint

      - name: Check for changes
        run: |
          git diff --exit-code schema/
          if [ $? -ne 0 ]; then
            echo "Schema is out of date. Run 'make schema' and commit."
            exit 1
          fi
```

## Embedding Schemas

Use Go's `embed` directive for runtime access:

```go
package schema

import _ "embed"

//go:embed config.schema.json
var ConfigSchema []byte

//go:embed config.schema.json
var ConfigSchemaString string
```

## Version Control

Commit both source and generated files:

```bash
git add config/config.go
git add schema/config.schema.json
git add docs/config-spec.md
git commit -m "feat: add configuration types with schema and docs"
```

This ensures the repository always has up-to-date artifacts.
