# AGENTS.md — schemakit

schemakit is a toolkit for working with JSON Schema in Go projects: it lints
schemas for static-type compatibility, generates schemas from Go structs, and
generates Markdown documentation from Go types.

## Conventions

- **Go module:** `github.com/grokify/schemakit`; the CLI is `cmd/schemakit`
  (the project was formerly named schemalint)
- **Library-first:** reusable packages (`linter`, `parser`, `schemakit`) with a
  thin Cobra CLI over them
- **Generated schemas:** Go structs are the source of truth; see
  `schemakit generate` and its `--check` drift guard

## Documentation Layout

| Path | Purpose | Update when |
|------|---------|-------------|
| `docs/index.md` | Site home and overview | Scope or entry points change |
| `docs/guides/` | User documentation: installation, command reference, reference pages, how-to guides | User-visible behavior changes (same change) |
| `docs/releases/` | Release notes (`vX.Y.Z.md`) and the changelog (`CHANGELOG.json`, `CHANGELOG.md`) | Each release |

- The changelog lives in `docs/releases/`, not the repository root, so MkDocs
  publishes it with the release notes. `CHANGELOG.json` is the source;
  regenerate the Markdown with
  `schangelog generate docs/releases/CHANGELOG.json -o docs/releases/CHANGELOG.md`.
- Every page must be listed in the `mkdocs.yml` nav. Check with
  `mkdocs build --strict`.

## Release Maintenance

1. Review commits since the previous tag: `schangelog parse-commits --since <tag>`.
2. Move `unreleased` entries (or add the new release) in
   `docs/releases/CHANGELOG.json`, validate with `schangelog validate`, and
   regenerate `docs/releases/CHANGELOG.md`.
3. Add `docs/releases/vX.Y.Z.md` and list it in the `mkdocs.yml` nav.
4. Run `go test ./...`, `golangci-lint run`, and `mkdocs build --strict`.
5. Check the release build before tagging, either locally with
   `goreleaser release --snapshot --clean --skip=publish,announce,validate`
   or in CI with `gh workflow run release-dry-run.yaml` (builds every target,
   uploads `dist/`, and previews the release notes in the job summary). The
   archive must contain the binary, README, LICENSE, and CHANGELOG.md.
6. Push, wait for CI, then tag `vX.Y.Z` and push the tag. The Release workflow
   calls the shared `grokify/.github` Go Release workflow, which runs
   GoReleaser and sets the release body: a link to `docs/releases/vX.Y.Z.md`
   on the docs site plus a compare link.
7. Confirm the GitHub release has the platform archives and `checksums.txt`.
   The shared workflow fails the run when a release has fewer than two assets.
