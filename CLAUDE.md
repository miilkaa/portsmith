# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## What This Repository Is

`portsmith` is a Go CLI toolkit for Clean Architecture packages. It is not an application and no longer ships runtime libraries.

It provides:

- `portsmith init` for `portsmith.yaml`
- `portsmith new` for package scaffolding
- `portsmith gen` for generated `ports.go`
- `portsmith mock` for mockery integration
- `portsmith check` for architecture linting

Generated code must not import Portsmith runtime helpers; this repository no longer ships them.

## Common Commands

```bash
go build ./...
go test ./...
go test ./internal/lint -run TestSpecificName
go test -v -run TestX/subcase ./internal/analyze

go run ./cmd/portsmith <subcommand> [args]
go run ./cmd/portsmith check ./...
go run ./cmd/portsmith version

go vet ./...
gofmt -w .
```

## Architecture Direction

Target structure:

```text
cmd/portsmith
  process entrypoint

internal/cli
  command selection, argument parsing, usage, version

internal/app
  command use cases

internal/project
  project root, go.mod, portsmith.yaml, stack resolution

internal/analyze
  source analysis shared by gen and lint

internal/lint
  lint engine and rules
```

## Current Rule Contract

Rule IDs are user-facing API. They appear in `portsmith.yaml` and `//nolint:portsmith:<rule>` comments. Keep the docs and implementation in sync when changing rules.

## Change Discipline

When the request is to review logic, run tests, or update tests around existing work, do not change production implementation unless explicitly asked. If an implementation change looks necessary in that situation, ask first and explain the proposed change before editing files.

## Documentation Conventions

- Go comments are in English.
- `README.md` is canonical; `README.ru.md` mirrors it.
- `docs/en/*.md` is canonical; `docs/ru/*.md` mirrors it.
- This repository does not contain examples or runtime packages.
