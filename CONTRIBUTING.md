# Contributing

Thanks for helping improve this project.

## Prerequisites

- Go **1.26.1** or newer (see `go.mod`).
- For a full local release dry-run: `git-cliff` (optional; used by `make changelog` / `make release`).

## Getting started

```bash
git clone https://github.com/bircni/workflow-check.git
cd workflow-check
```

Run the CLI from the working tree without installing:

```bash
go run ./cmd/workflow-lock version
```

Or build a binary:

```bash
make build
./bin/workflow-lock version
```

## Checks before you open a PR

CI runs the same steps as:

```bash
make verify
make lint
make test
make coverage
```

Fix formatting and style issues with:

```bash
make fmt
make lint
```

## Commits and changelog

This repo uses [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, etc.). They feed [git-cliff](https://git-cliff.org/) and `CHANGELOG.md`, so please use a type that matches your change.

## Questions

Open an issue if something is unclear or if you want to discuss a larger change before writing code.
