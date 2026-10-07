# Installation

## Go Install

```bash
go install github.com/grokify/schemakit/cmd/schemakit@latest
```

## Release Binaries

Prebuilt archives for Linux, macOS, and Windows (amd64 and arm64) are attached to each
[GitHub release](https://github.com/grokify/schemakit/releases), starting with v0.6.0.
Each archive contains the `schemakit` binary, the README, the license, and the changelog.
Download the one for your platform, extract it, and put `schemakit` on your `PATH`.

## From Source

```bash
git clone https://github.com/grokify/schemakit.git
cd schemakit
go build -o schemakit ./cmd/schemakit
```

## Verify Installation

```bash
schemakit version
```
