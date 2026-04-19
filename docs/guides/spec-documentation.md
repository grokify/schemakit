# Creating Human-Audience Specification Documents

This guide explains how to use schemakit to create specification documents for human readers, and which parts can be auto-generated vs. require human authoring.

## Overview

schemakit can auto-generate approximately **90%** of a specification document from Go source code. The remaining 10% benefits from human authoring to provide context, examples, and external references.

## What Gets Auto-Generated

| Content | Source | Quality |
|---------|--------|---------|
| Type definitions | Go struct types | Excellent |
| Field tables | Struct fields + JSON tags | Excellent |
| Required/Optional | `omitempty` tag | Excellent |
| Type descriptions | Type doc comments | Good (if comments exist) |
| Field descriptions | Field doc comments | Good (if comments exist) |
| Enum types | `type X string` declarations | Excellent |
| Enum values | `const` blocks | Excellent |
| Enum descriptions | Const doc comments | Variable |

## What Benefits from Human Authoring

| Content | Why Human Authoring Helps |
|---------|---------------------------|
| **Overview section** | Explains purpose, design philosophy, when to use |
| **JSON examples** | Shows realistic usage, not just field listings |
| **External references** | Links to MITRE ATT&CK, OWASP, RFCs, etc. |
| **Conceptual grouping** | Organizes types by domain, not discovery order |
| **Usage guidance** | "When to use X vs Y" decisions |
| **Diagrams** | Architecture, data flow visualizations |

## Recommended Workflow

### Step 1: Create a Header File

Create `header.md` with human-authored content:

```markdown
# My Specification v1.0.0

## Overview

This specification defines the data model for...

## Key Concepts

- **Concept A**: Explanation...
- **Concept B**: Explanation...

## Examples

### Basic Example

```json
{
  "type": "example",
  "name": "My Example"
}
```

## References

- [External Spec](https://example.com)
- [Related Standard](https://example.com)

---

```

Note the `---` at the end to separate from auto-generated content.

### Step 2: Generate the Spec

```bash
schemakit doc --prepend header.md \
  -t "My Specification" -v v1.0.0 \
  github.com/myorg/myproject/types RootType \
  -o specification.md
```

### Step 3: Review and Iterate

1. Check that type descriptions are clear
2. Add doc comments to Go source where descriptions show "-"
3. Regenerate to incorporate improvements

## Best Practices for Go Source

### Add Doc Comments to Types

```go
// ThreatModel is the root type containing all threat modeling data.
// It supports multiple diagram views and shared security mappings.
type ThreatModel struct {
    // ID is a unique identifier for this threat model.
    ID string `json:"id"`

    // Title is a human-readable name for the threat model.
    Title string `json:"title"`
}
```

### Add Doc Comments to Enum Values

```go
type ThreatStatus string

const (
    // ThreatStatusIdentified indicates the threat has been identified.
    ThreatStatusIdentified ThreatStatus = "identified"

    // ThreatStatusMitigated indicates the threat has been mitigated.
    ThreatStatusMitigated ThreatStatus = "mitigated"
)
```

### Use Meaningful JSON Tags

```go
// Good: clear, consistent naming
Title string `json:"title"`
CreatedAt time.Time `json:"createdAt,omitempty"`

// Avoid: inconsistent or unclear
Title string `json:"t"`
```

## Example: threat-model-spec

The [threat-model-spec](https://github.com/grokify/threat-model-spec) project uses this workflow:

**Auto-generated content:**
- 38 struct types with field tables
- 27 enum types with value tables
- Type and field descriptions from doc comments

**Human-authored content (in header):**
- Overview explaining the specification purpose
- JSON examples for each diagram type (DFD, Attack Chain, etc.)
- External references (MITRE ATT&CK, OWASP, STRIDE)
- Framework mapping tables

**Generation command:**

```bash
schemakit doc --prepend docs/spec-header.md \
  -t "Threat Model Specification" -v v0.4.0 \
  github.com/grokify/threat-model-spec/ir ThreatModel \
  -o docs/versions/v0.4.0/specification.md
```

## Versioning Specifications

For versioned specifications (like OpenAPI):

```
docs/
├── versions/
│   ├── v0.3.0/
│   │   └── specification.md
│   └── v0.4.0/
│       ├── specification.md      # Generated
│       └── threat-model.schema.json
└── spec-header.md                # Shared header template
```

Update the version in both the header and the `-v` flag when releasing.

## Automation

Add to your Makefile or CI:

```makefile
.PHONY: docs
docs:
	schemakit doc --prepend docs/header.md \
	  -t "My Specification" -v $(VERSION) \
	  github.com/myorg/myproject/types RootType \
	  -o docs/specification.md
```

## Summary

| Approach | Auto-Gen % | Best For |
|----------|------------|----------|
| Pure auto-gen | 100% | Internal API docs, quick reference |
| Header + auto-gen | ~90% | Public specifications, user guides |
| Fully manual | 0% | Marketing docs, tutorials |

The `--prepend` approach gives you the best of both worlds: reproducible auto-generated content with human-crafted context.
