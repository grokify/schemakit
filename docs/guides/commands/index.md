# Commands

schemakit provides three main commands:

| Command | Description |
|---------|-------------|
| [`lint`](lint.md) | Check schemas for static type compatibility |
| [`generate`](generate.md) | Generate JSON Schema from Go struct types |
| [`doc`](doc.md) | Generate Markdown documentation from Go types |

## Common Patterns

### Go-First Development

```bash
# 1. Define types in Go with doc comments
# 2. Generate schema
schemakit generate github.com/myorg/myproject/types Config -o schema.json

# 3. Generate documentation
schemakit doc github.com/myorg/myproject/types Config -o spec.md

# 4. Validate schema
schemakit lint schema.json
```

### Schema-First Development

```bash
# 1. Write or receive a JSON Schema
# 2. Lint before code generation
schemakit lint schema.json --profile scale

# 3. Generate code (using external tools)
```
