# Linting Profiles

schemakit supports multiple linting profiles for different use cases.

## Available Profiles

| Profile | Use Case |
|---------|----------|
| `default` | General schema validation |
| `scale` | Strict mode for static type generation |
| `navigable` | Human-reviewable, AI-friendly schemas |

## Default Profile

The default profile checks for common patterns that cause problems in code generation while allowing standard JSON Schema features.

```bash
schemakit lint schema.json
schemakit lint schema.json --profile default
```

### Checks

- Union discriminator validation
- Large union warnings
- Nested union warnings
- Property case conventions

### When to Use

- General-purpose schemas
- Schemas that need `anyOf`/`oneOf` for flexibility
- Gradual migration to stricter patterns

## Scale Profile

The scale profile enforces strict rules for clean, unambiguous code generation in statically-typed languages.

```bash
schemakit lint schema.json --profile scale
```

### Additional Checks

| Check | Rationale |
|-------|-----------|
| No `anyOf`/`oneOf`/`allOf` | These map poorly to static types |
| No `additionalProperties: true` | Creates `map[string]any` types |
| Require explicit `type` | Prevents ambiguous inference |
| No mixed types | `["string", "number"]` creates union types |

### When to Use

- Schemas designed for Go, Rust, TypeScript
- Maximum code generation compatibility
- Strict type safety requirements

## Navigable Profile

The navigable profile enforces patterns that make schemas easy to review by humans and author by AI agents. Based on design principles from [incident-lifecycle-spec](https://github.com/plexusone/incident-lifecycle-spec).

```bash
schemakit lint schema.json --profile navigable
```

### Design Principles

1. **Flat structure** — Maximum 2 levels of object nesting
2. **Single-hop references** — Cross-references should be shallow and predictable
3. **ID fields** — Array items should have ID fields for cross-referencing
4. **Locally comprehensible** — Each object can be understood without context

### Checks

| Check | Severity | Rationale |
|-------|----------|-----------|
| Object nesting > 2 levels | Error | Deep nesting is hard to navigate |
| Arrays of arrays of objects | Warning | Reduces navigability |
| Array items without ID field | Warning | Prevents cross-referencing |

### When to Use

- Schemas designed for human review (postmortems, incidents)
- Schemas authored by AI agents with human-in-the-loop review
- Documentation-first schemas where structure matters
- Any schema where you say "I'll figure out what this means later"

### Example: Good Navigable Schema

```json
{
  "type": "object",
  "properties": {
    "timeline": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "event_id": { "type": "string" },
          "description": { "type": "string" },
          "evidence_ids": {
            "type": "array",
            "items": { "type": "string" }
          }
        }
      }
    },
    "evidence": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "evidence_id": { "type": "string" },
          "description": { "type": "string" }
        }
      }
    }
  }
}
```

This schema uses flat top-level arrays with ID fields, enabling cross-references like `evidence_ids: ["evi-001"]` instead of nested evidence objects.

## Choosing a Profile

```
┌─────────────────────────────────────────────────────┐
│                   Your Schema                        │
└─────────────────────┬───────────────────────────────┘
                      │
    ┌─────────────────┼─────────────────┐
    │                 │                 │
 Need unions?    Human/AI review?   No unions?
    │                 │                 │
    ▼                 ▼                 ▼
┌─────────┐     ┌───────────┐     ┌─────────┐
│ default │     │ navigable │     │  scale  │
└─────────┘     └───────────┘     └─────────┘
```

## Profile Comparison

| Feature | default | scale | navigable |
|---------|---------|-------|-----------|
| `anyOf` | ✅ Allowed | ❌ Error | ✅ Allowed |
| `oneOf` | ✅ Allowed | ❌ Error | ✅ Allowed |
| `allOf` | ✅ Allowed | ❌ Error | ✅ Allowed |
| `additionalProperties: true` | ⚠️ Warning | ❌ Error | ⚠️ Warning |
| Mixed type arrays | ✅ Allowed | ❌ Error | ✅ Allowed |
| Missing `type` | ✅ Allowed | ❌ Error | ✅ Allowed |
| Large unions | ⚠️ Warning | ⚠️ Warning | ⚠️ Warning |
| Property case | ✅ Checked | ✅ Checked | ✅ Checked |
| Deep object nesting | ✅ Allowed | ✅ Allowed | ❌ Error (>2) |
| Array items without ID | ✅ Allowed | ✅ Allowed | ⚠️ Warning |

## Custom Configuration

Currently, profiles are predefined. Future versions may support custom profile configuration via a config file.

## Migration Path

To migrate from default to scale:

1. Run with `--profile scale`
2. Address each error:
   - Replace `anyOf`/`oneOf` with explicit types
   - Add explicit `type` fields
   - Remove `additionalProperties: true`
3. Re-run until clean
