# Changelog

See the full [CHANGELOG.md](https://github.com/grokify/schemakit/blob/main/CHANGELOG.md) on GitHub.

## Recent Releases

### v0.4.0 (2026-04-18)

**Highlights:** Rename to schemakit and add doc command for Markdown documentation generation

**Breaking Changes:**

- Rename binary from `schemalint` to `schemakit`
- Rename module from `github.com/grokify/schemalint` to `github.com/grokify/schemakit`

**Added:**

- `schemakit doc` command to generate Markdown documentation from Go struct types
- Extract package doc comments, type descriptions, and field comments via reflection
- Generate required/optional field tables with JSON names and types
- Support `--title` and `--version` flags for documentation headers
- Enum type extraction from `type X string` declarations
- Enum value extraction from `const` blocks with descriptions
- `--prepend` flag for including custom header content

### v0.3.0 (2026-02-09)

**Highlights:** Generate JSON Schema from Go struct types using invopop/jsonschema

**Added:**

- `schemalint generate` command to generate JSON Schema from Go struct types
- Support for both local (GOPATH/src) and remote Go modules in schema generation
- GitHub Actions release workflow for GoReleaser

### v0.2.0 (2026-02-08)

**Highlights:** Rename to schemalint and add scale profile for strict static type compatibility

**Breaking Changes:**

- Rename binary from `schemago` to `schemalint`
- Rename module from `github.com/grokify/schemago` to `github.com/grokify/schemalint`

**Added:**

- `--profile` / `-p` flag to select linting profile (`default`, `scale`)
- Scale profile with strict rules for static type generation
- `--property-case` flag to enforce property naming conventions
- Goreleaser configuration for cross-platform builds and Homebrew tap

### v0.1.0 (2026-01-17)

**Highlights:** Initial release with JSON Schema linter for Go compatibility

**Added:**

- CLI with `lint` and `version` commands
- JSON Schema parser for linting
- Union detection with discriminator field analysis
- Multiple output formats: text, JSON, GitHub Actions
