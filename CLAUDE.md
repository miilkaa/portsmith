# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repository is

`portsmith` is itself a Go framework + CLI for building Clean-Architecture backends — **not** an application. It ships:

- Runtime libraries under `pkg/` that downstream apps import (`apperrors`, `config`, `database`, `server`, `chiserver`, `sqlxdb`, `pagination`, `testkit`).
- A CLI under `cmd/portsmith` with subcommands `init`, `gen`, `new`, `mock`, `check`.
- Reference example packages under `examples/` that are **embedded into the binary** via `assets.go` (`go:embed examples`). Removing or renaming `examples/` will break the build.

When editing code here, you are usually editing the framework / CLI itself, not user-facing application code.

## Common commands

There is no Makefile. Use the Go toolchain directly.

```bash
# Build and tests
go build ./...
go test ./...
go test ./internal/lint -run TestSpecificName    # one test
go test -v -run TestX/subcase ./internal/gen     # one subtest

# Run the CLI from source
go run ./cmd/portsmith <subcommand> [args]
go run ./cmd/portsmith check ./...               # lint this very repo
go run ./cmd/portsmith version

# Lint / vet
go vet ./...
gofmt -w .
```

The CLI uses standard library `flag`-style argument parsing; `cmd/portsmith/main.go` dispatches by `os.Args[1]`. To add a new subcommand, add a sibling under `cmd/portsmith/<name>/` exposing `Run(args []string) error` and wire the `case` in `main.go`.

## Architecture: how the pieces fit

### CLI layout

`cmd/portsmith/main.go` is a thin dispatcher. Each subcommand lives in its own package and delegates to `internal/`:

| Subcommand | Calls into |
|---|---|
| `init` | `cmd/portsmith/init` (interactive wizard, writes `portsmith.yaml`) |
| `gen` | `internal/gen` (AST + regex extraction → writes `ports.go`) |
| `new` | `cmd/portsmith/new` (scaffolding; templates depend on resolved stack) |
| `mock` | wraps `mockery` |
| `check` | `internal/lint` + `internal/lintconfig` |

### Stack resolution (`internal/stack`)

Two stacks are supported: `gin-gorm` (default) and `chi-sqlx`. Resolution priority — **same in `new` and `check`**:

1. `--stack` flag
2. `stack:` field in `portsmith.yaml` at project root
3. `go.mod` heuristic — `go-chi/chi` → `chi-sqlx`, `gin-gonic/gin` → `gin-gorm`
4. Fallback to `gin-gorm`

Project root = walk up from a target dir until a directory with `go.mod` is found (`stack.FindProjectRoot`). Use `stack.Resolve(pkgDir, stackFlag)` for a per-package CLI command, `stack.ResolveFromWD(stackFlag)` for one tied to the working directory.

### Lint engine (`internal/lint` + `internal/lintconfig`)

Entry point: `lint.Violations(dir, cfg, projectRoot) ([]Violation, error)`.

Each rule is one file `rule_*.go` and one function `checkXxx(ctx CheckContext) []Violation`. To add a rule:

1. Create `internal/lint/rule_<name>.go` exposing `checkX(ctx CheckContext) []Violation`.
2. Append it inside `checkFile` in `internal/lint/lint.go` (or top-level alongside `checkPortsPresence` for non-per-file checks).
3. Use a stable `Rule` id string — that id is what users put in `lint.rules.<id>.severity` and in `//nolint:portsmith:<id>` suppressions.
4. If the rule is opt-in, gate it on a field in `lintconfig.LintConfig` (see `LoggerConfig`, `CallPatternsConfig` for the pattern: `Enabled()` method on the config + early return).
5. Severity is filtered centrally in `filterRulesOff` (off) and in `cmd/portsmith/check/check.go` (warning vs error). Inline `//nolint:portsmith:<rule>` is handled in `internal/lint/suppress.go`.

The full set of rule IDs and what they enforce is documented in `docs/en/architecture.md` (and its Russian twin) — keep that table and the actual rule files in sync when changing IDs.

### Generator (`internal/gen`)

`portsmith gen` produces a **minimal** `ports.go` from real usage. It combines:

- AST parsing to extract method signatures from `service.go` / `repository.go`.
- Regex-based call collection (`repoCallRe`, `svcCallRe`, alias patterns) to know which methods to expose.

The interface name prefix is detected from existing `Handler` / `Service` field types; the folder name is only the fallback. When changing the regexes, run `go test ./internal/gen` — `gen_test.go` covers the alias/exported-method matching cases.

### Runtime packages (`pkg/`)

Two parallel HTTP/DB toolchains exist deliberately:

- gin-gorm stack → `pkg/server` + `pkg/database`
- chi-sqlx stack → `pkg/chiserver` + `pkg/sqlxdb`

Both expose comparable surface area (server with `/health`, recovery, request-id, CORS; DB connect + transactions). Keep them in feature parity when you add anything user-facing to one side.

`pkg/apperrors` is the single source of truth for error → HTTP-status mapping; both servers consume it.

`pkg/testkit` is shared and uses SQLite in-memory for repository tests regardless of stack.

## Project conventions

### Bilingual content (strict — see `.cursor/rules/bilingual-docs.mdc`)

- **All Go comments must be in English.** No Russian in `.go` files.
- `docs/en/*.md` is canonical; `docs/ru/*.md` mirrors it. Updating one without the other is a doc bug.
- `README.md` is canonical; `README.ru.md` mirrors it.
- `examples/` always contains parallel `_en` and `_ru` packages. When adding a new example package, create both variants.

### Self-application

This repo is also a portsmith user — `go run ./cmd/portsmith check ./...` should pass on `main`. If a change to a lint rule starts failing on this repo's own code (e.g. `internal/lint`, `pkg/`), prefer fixing the rule logic or the targeted file over disabling the rule globally. Wiring methods (`Set*` / `With*`) are explicitly exempt from `context-first` — see `df304ce`.

### Embedded examples

`assets.go` does `//go:embed examples`. Two consequences:

- The `examples/` directory must exist and contain valid Go (the `_en` and `_ru` packages).
- Test or refactor changes to example packages can break `go build` of the root module.

### Naming the lint rule IDs

Rule IDs are user-facing (used in YAML and `//nolint`). Treat them as a stable API: don't rename without a deprecation. The current IDs are listed in `docs/en/architecture.md` "Core rules" and the call/logger sections — that table is the contract.