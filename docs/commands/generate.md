# schemakit generate

Generate JSON Schema from Go struct types using reflection.

## Usage

```bash
schemakit generate <package> <type> [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output file (default: stdout) |
| `--indent` | Indent JSON output (default: true) |
| `--check` | Verify the committed `-o` file matches freshly generated output; exit non-zero on drift (no write). Requires `-o`. |
| `--comments` | Use Go doc comments from the target package (and its subdirectories) as property and type descriptions. |
| `--id` | Set the schema's `$id` (default: the package path plus the lower-cased type name). |
| `--title` | Set the schema's `title`. |
| `--description` | Set the schema's `description`. |

## Examples

```bash
# Generate to stdout
schemakit generate github.com/myorg/myproject/types Config

# Save to file
schemakit generate -o schema.json github.com/myorg/myproject/types Config

# Without indentation
schemakit generate --indent=false github.com/myorg/myproject/types Config

# CI drift guard: fail if the committed schema is out of sync with the Go structs
schemakit generate -o schema.json --check github.com/myorg/myproject/types Config

# Document properties from Go doc comments and set schema metadata
schemakit generate --comments \
  --id https://example.com/schemas/config.schema.json \
  --title "Config" --description "Application configuration." \
  -o schema.json github.com/myorg/myproject/types Config
```

## Descriptions from Go Comments (`--comments`)

With `--comments`, Go doc comments become `description` fields, so the Go
source stays the single place where the contract is documented:

```go
// Config configures the service.
type Config struct {
	// Host is the database host name.
	Host string `json:"host"`
}
```

- A **type** comment contributes its first sentence; a **field** comment
  contributes its full text.
- On a field, a `jsonschema:"description=X"` tag takes precedence over the
  comment.
- Comments are read from the target package's directory and below. Types
  from other packages are reflected but get no descriptions.
- Every `.go` file under that directory is parsed, so a syntactically
  invalid file (for example one under `testdata`) makes generation fail.
- Comments are read from source, so the module must be available locally
  (under `$GOPATH/src`) or in the module cache.

## Schema Metadata (`--id`, `--title`, `--description`)

These flags set the corresponding top-level keywords on the generated
schema. Values are quoted before they are placed in the generated program,
so quotes, newlines, and backslashes are safe. Use `--id` when the schema
is published at a stable URL, such as a raw repository file, rather than
the default derived from the package path.

## Drift Guard (`--check`)

`--check` regenerates the schema in memory and compares it to the committed
file at `-o`, exiting non-zero on any difference without rewriting the file.
This keeps a committed schema honest against its Go source of truth. Pair it
with a `//go:generate` directive so `go generate ./...` refreshes the schema
and CI enforces it:

```go
//go:generate schemakit generate -o schema.json github.com/myorg/myproject/types Config
```

```bash
# In CI
schemakit generate -o schema.json --check github.com/myorg/myproject/types Config
```

## How It Works

The command creates a temporary Go program that:

1. Imports your target package
2. Uses [invopop/jsonschema](https://github.com/invopop/jsonschema) to reflect on the type
3. Generates a JSON Schema with `$defs` for nested types

## Module Resolution

The command supports both local and remote packages:

- **Local modules**: Packages in `$GOPATH/src` are resolved via replace directives
- **Remote modules**: Packages are fetched via `go get`

## Struct Tags

The generated schema respects these struct tags:

| Tag | Effect |
|-----|--------|
| `json:"name"` | Sets the property name |
| `json:",omitempty"` | Marks field as optional |
| `json:"-"` | Excludes field from schema |
| `jsonschema:"title=X"` | Sets schema title |
| `jsonschema:"description=X"` | Sets schema description |
| `jsonschema:"enum=a,b,c"` | Defines enum values |

## Example Output

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/myorg/myproject/types/config",
  "$defs": {
    "Database": {
      "type": "object",
      "properties": {
        "host": { "type": "string" },
        "port": { "type": "integer" }
      },
      "required": ["host", "port"]
    }
  },
  "type": "object",
  "properties": {
    "database": { "$ref": "#/$defs/Database" }
  }
}
```
