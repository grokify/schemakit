# Linting Profiles

schemakit supports multiple linting profiles for different use cases.

## Available Profiles

| Profile | Use Case |
|---------|----------|
| `default` | General schema validation |
| `scale` | Strict mode for static type generation |

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

## Choosing a Profile

```
┌─────────────────────────────────────────────────────┐
│                   Your Schema                        │
└─────────────────────┬───────────────────────────────┘
                      │
         ┌────────────┴────────────┐
         │                         │
    Need unions?              No unions?
         │                         │
         ▼                         ▼
   ┌───────────┐            ┌───────────┐
   │  default  │            │   scale   │
   └───────────┘            └───────────┘
```

## Profile Comparison

| Feature | default | scale |
|---------|---------|-------|
| `anyOf` | ✅ Allowed (with discriminator) | ❌ Error |
| `oneOf` | ✅ Allowed (with discriminator) | ❌ Error |
| `allOf` | ✅ Allowed | ❌ Error |
| `additionalProperties: true` | ⚠️ Warning | ❌ Error |
| Mixed type arrays | ✅ Allowed | ❌ Error |
| Missing `type` | ✅ Allowed | ❌ Error |
| Large unions | ⚠️ Warning | ⚠️ Warning |
| Property case | ✅ Checked | ✅ Checked |

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
