# Technical Requirements Document: schemalint

## 1. Overview

This document describes the technical architecture and implementation requirements for **schemalint**, a JSON Schema linter for static type compatibility.

### 1.1 Context

JSON Schema (Draft 2020-12) is widely used to define data models for APIs and specifications. However, many schema patterns don't map cleanly to statically-typed languages, causing code generators to produce suboptimal output (e.g., `interface{}` in Go, `any` in TypeScript).

### 1.2 Scope

This document covers:

- System architecture and component design
- Linting rules and profiles
- CLI interface design
- Testing requirements

## 2. Architecture

### 2.1 High-Level Pipeline

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  JSON Schema    │────▶│  Schema Parser   │────▶│  Linter         │
│  (input.json)   │     │                  │     │                 │
└─────────────────┘     └──────────────────┘     └────────┬────────┘
                                                          │
                                                          ▼
                                               ┌──────────────────┐
                                               │  Result          │
                                               │  - Issues        │
                                               │  - Suggestions   │
                                               └────────┬─────────┘
                                                        │
                                                        ▼
                                               ┌──────────────────┐
                                               │  Output          │
                                               │  - Text/JSON     │
                                               │  - GitHub        │
                                               └──────────────────┘
```

### 2.2 Package Structure

```
schemalint/
├── cmd/
│   └── schemalint/           # CLI entry point            ✅ Implemented
│       └── main.go
├── linter/                   # Schema linting             ✅ Implemented
│   ├── linter.go             # Core linting logic
│   ├── linter_test.go        # Unit tests
│   ├── schema.go             # JSON Schema types
│   └── issue.go              # Issue/Result types
├── testdata/                 # Test schemas               ✅ Implemented
│   ├── good_schema.json
│   ├── bad_schema.json
│   ├── scale_valid.json
│   └── scale_invalid.json
└── .goreleaser.yaml          # Release configuration      ✅ Implemented
```

## 3. Linter Package

The `linter/` package provides schema validation for static type compatibility.

### 3.1 Profiles

| Profile | Description |
|---------|-------------|
| `default` | Standard checks with errors and warnings |
| `scale` | Strict mode that disallows composition keywords |

### 3.2 Issue Codes (Default Profile)

| Code | Severity | Description |
|------|----------|-------------|
| `union-no-discriminator` | Error | Union lacks discriminator field |
| `inconsistent-discriminator` | Error | Variants use different discriminators |
| `missing-const` | Error | Variant lacks const value |
| `duplicate-const-value` | Error | Multiple variants have same value |
| `large-union` | Warning | Union has >10 variants |
| `nested-union` | Warning | Union nested >2 levels deep |
| `additional-properties` | Warning | Variant has `additionalProperties: true` |

### 3.3 Issue Codes (Scale Profile)

| Code | Severity | Description |
|------|----------|-------------|
| `composition-disallowed` | Error | `anyOf`, `oneOf`, `allOf` used |
| `additional-properties-disallowed` | Error | `additionalProperties: true` |
| `missing-type` | Error | No explicit `type` field |
| `mixed-type-disallowed` | Error | Type array like `["string", "number"]` |

### 3.4 Configuration

```go
type Profile string

const (
    ProfileDefault Profile = "default"
    ProfileScale   Profile = "scale"
)

type Config struct {
    Profile              Profile
    MaxUnionVariants     int      // Default: 10
    MaxUnionNestingDepth int      // Default: 2
    DiscriminatorFields  []string // Default: ["component_type", "type", "kind"]
}
```

### 3.5 Pattern Detection

The linter correctly identifies and handles:

- **Nullable patterns**: `anyOf: [T, null]` - Not flagged as missing discriminator
- **Reference patterns**: `anyOf: [ComponentReference, BaseXxx]` - Recognized by `$component_ref` property
- **All-$ref unions**: Unions where all variants are `$ref` - Skipped (requires resolution)

### 3.6 Type Array Handling

The Schema struct handles both single types and type arrays:

```go
type Schema struct {
    Type     string   // Single type (e.g., "object")
    TypeList []string // Type array (e.g., ["string", "null"])
    // ...
}

func (s *Schema) HasMixedType() bool {
    return len(s.TypeList) > 1
}

func (s *Schema) HasType() bool {
    return s.Type != "" || len(s.TypeList) > 0
}
```

## 4. CLI Interface

### 4.1 Commands

```bash
# Lint with default profile
schemalint lint schema.json

# Lint with scale profile
schemalint lint --profile scale schema.json

# Output formats
schemalint lint --output text schema.json   # Human-readable (default)
schemalint lint --output json schema.json   # Machine-readable JSON
schemalint lint --output github schema.json # GitHub Actions annotations

# Version
schemalint version
```

### 4.2 Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No issues found |
| 1 | Errors found |
| 2 | Warnings found (no errors) |

### 4.3 Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `text` | Output format: text, json, github |
| `--profile` | `-p` | `default` | Linting profile: default, scale |

## 5. Testing Requirements

### 5.1 Unit Tests

| Component | Coverage Target | Current |
|-----------|-----------------|---------|
| Linter (default profile) | 90% | ✅ Implemented |
| Linter (scale profile) | 90% | ✅ Implemented |
| Schema parsing | 90% | ✅ Implemented |
| Result formatting | 80% | ✅ Implemented |

### 5.2 Test Cases

Default profile tests:

- `TestLintNullablePattern` - Nullable patterns skipped
- `TestLintUnionWithDiscriminator` - Valid discriminated unions pass
- `TestLintUnionWithoutDiscriminator` - Missing discriminator flagged
- `TestLintLargeUnion` - Large union warning
- `TestLintAdditionalProperties` - additionalProperties warning
- `TestLintAllRefs` - All-$ref unions skipped

Scale profile tests:

- `TestScaleProfileDisallowsAnyOf` - anyOf flagged
- `TestScaleProfileDisallowsOneOf` - oneOf flagged
- `TestScaleProfileDisallowsAllOf` - allOf flagged
- `TestScaleProfileDisallowsAdditionalProperties` - additionalProperties flagged
- `TestScaleProfileRequiresType` - Missing type flagged
- `TestScaleProfileDisallowsMixedTypes` - Type arrays flagged
- `TestDefaultProfileAllowsComposition` - Regression test
- `TestScaleProfileValidSchema` - Valid schema passes

### 5.3 Test Schemas

- `testdata/good_schema.json` - Valid schema with discriminators
- `testdata/bad_schema.json` - Schema with default profile issues
- `testdata/scale_valid.json` - Schema that passes scale profile
- `testdata/scale_invalid.json` - Schema with scale profile violations

## 6. Dependencies

### 6.1 Runtime Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/spf13/cobra` | CLI framework |

### 6.2 Development Dependencies

| Dependency | Purpose |
|------------|---------|
| `golangci-lint` | Code linting |
| `goreleaser` | Release automation |

## 7. Distribution

### 7.1 Platforms

| OS | Arch |
|----|------|
| linux | amd64, arm64 |
| darwin | amd64, arm64 |
| windows | amd64, arm64 |

### 7.2 Installation Methods

```bash
# Homebrew
brew install grokify/tap/schemalint

# Go install
go install github.com/grokify/schemalint/cmd/schemalint@latest
```

### 7.3 Release Process

1. Update docs/releases/CHANGELOG.json and regenerate docs/releases/CHANGELOG.md
2. Commit and push to main
3. Wait for CI to pass
4. Create and push tag: `git tag v0.x.0 && git push origin v0.x.0`
5. GoReleaser builds binaries and updates Homebrew tap

## 8. Implementation Status

### Phase 1: Foundation ✅ Complete (v0.1.0)

- [x] Project structure and CI setup
- [x] CLI with cobra (`lint`, `version` commands)
- [x] JSON Schema parsing
- [x] Union detection algorithm
- [x] Nullable/reference pattern detection
- [x] Discriminator detection
- [x] Multiple output formats
- [x] Unit tests

### Phase 2: Scale Profile ✅ Complete (v0.2.0)

- [x] Profile configuration
- [x] Composition keyword checks
- [x] additionalProperties check
- [x] Missing type check
- [x] Mixed type array check
- [x] Type array parsing
- [x] Scale profile tests
- [x] GoReleaser configuration
- [x] Documentation updates

### Phase 3: Polish 🔲 Planned

- [ ] Configuration file support
- [ ] Additional lint checks
- [ ] $ref resolution
- [ ] Performance optimization

## 9. References

- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)
- [Go JSON Marshaling](https://pkg.go.dev/encoding/json)
- [Cobra CLI Framework](https://github.com/spf13/cobra)
- [GoReleaser](https://goreleaser.com/)
