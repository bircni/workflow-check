# Agent notes (workflow-check)

This repository is **Go only**: the `workflow-lock` CLI lives under `cmd/workflow-lock`; core logic is in `internal/workflowlock` (discovery, normalization, lockfile, `git ls-remote` resolution) and `internal/app` (flags, config file, commands). Version string embedding uses `internal/appmeta` and `-ldflags` from the `Makefile`.

## Commands

Typical targets:

- **`make fmt`** — gofumpt (`./cmd`, `./internal`)
- **`make lint`** — gofumpt check (no diff) + golangci-lint (see `.golangci.yml`)
- **`make test`** — `go test ./...`
- **`make verify`** — run `workflow-lock verify` on the repo
- **`make coverage`** — enforces total coverage threshold (see `COVERAGE_THRESHOLD` in `Makefile`)
- **`make ci`** — lint, test, coverage, verify (mirrors CI intent)
- **`make build`** — `bin/workflow-lock` with version from `VERSION`

After changing **`go.mod` / `go.sum`**, run **`go mod tidy`** (there is no `make tidy`).

## Project conventions

- Prefer small, focused changes; match existing style and imports.
- **Preserve** existing comments; do not delete or rewrite them unless they are wrong or obsolete.
- Avoid trailing whitespace in edited files.
- **Conventional Commits** and changelog expectations: see [CONTRIBUTING.md](CONTRIBUTING.md).
- User-facing behavior and install paths: [README.md](README.md).
- Add **authorship** or **`Co-Authored-By`** trailer lines only when the user or project maintainer asks for them.
