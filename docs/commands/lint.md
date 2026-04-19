# schemakit lint

Check a JSON Schema for patterns that cause problems in code generation.

## Usage

```bash
schemakit lint <schema.json> [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format: `text` (default), `json`, `github` |
| `-p, --profile` | Linting profile: `default`, `scale` |
| `--property-case` | Property case convention: `none`, `camelCase`, `snake_case`, `kebab-case`, `PascalCase` |

## Examples

```bash
# Basic lint
schemakit lint schema.json

# Use strict scale profile
schemakit lint schema.json --profile scale

# JSON output for CI
schemakit lint schema.json --output json

# GitHub Actions annotations
schemakit lint schema.json --output github

# Enforce snake_case properties
schemakit lint schema.json --property-case snake_case
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No issues found |
| 1 | Errors found (schema has problems) |
| 2 | Warnings found but no errors |

## Profiles

### Default Profile

Standard checks for code generation compatibility:

- Union without discriminator fields
- Inconsistent discriminator field names
- Missing const values in union variants
- Large unions (>10 variants)
- Deeply nested unions

### Scale Profile

Strict mode that disallows composition keywords:

- All default checks, plus:
- Disallow `anyOf`, `oneOf`, `allOf`
- Disallow `additionalProperties: true`
- Require explicit `type` field
- Disallow mixed type arrays

See [Lint Checks](../reference/lint-checks.md) for the complete list.
