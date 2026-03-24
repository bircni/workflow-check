# workflow-lock

`workflow-lock` keeps workflow files readable while locking remote action refs to immutable commit SHAs in `workflow-lock.yaml`.

The current tool version is tracked in [VERSION](/Users/nicolas/Github/bircni/workflow-check/VERSION) and exposed by `workflow-lock version`.

It supports GitHub-style and Gitea-style refs, including:

- `owner/repo@ref`
- `owner/repo/.github/workflows/file.yml@ref`
- `github.com/owner/repo@ref`
- `gitea.example.com/owner/repo@ref`

Ignored in v1:

- local `./...` refs
- generic URL forms such as `https://...`

## Usage

```bash
go run ./cmd/workflow-lock lock
go run ./cmd/workflow-lock verify
go run ./cmd/workflow-lock list
go run ./cmd/workflow-lock diff
go run ./cmd/workflow-lock version
```

Defaults:

- workflows: `.github/workflows`
- lockfile: `workflow-lock.yaml`
- default host: `github.com`

Useful flags:

- `-default-host code.gitea.example.com` to treat plain `owner/repo@ref` entries as coming from a self-hosted Gitea instance
- `-format json` for machine-readable `list` and `diff`
- `-config .workflow-lock-config.yaml` to load repo-level defaults

Examples:

```bash
go run ./cmd/workflow-lock lock -default-host code.gitea.example.com
go run ./cmd/workflow-lock diff -format json
go run ./cmd/workflow-lock verify -config .workflow-lock-config.yaml
```

## Config

If `.workflow-lock-config.yaml` exists in the repo root, the CLI loads it automatically. Flags still override config values.

Example:

```yaml
workflows: .github/workflows
lockfile: workflow-lock.yaml
default_host: github.com
```

A ready-to-copy example lives at [.workflow-lock-config.example.yaml](/Users/nicolas/Github/bircni/workflow-check/.workflow-lock-config.example.yaml).

## Development

```bash
make fmt
make lint
make test
make coverage
make verify
make build
make dist
make changelog
make release
make clean
```

## Releases

- `make dist` builds local cross-platform binaries and writes `dist/SHA256SUMS`.
- `make coverage` enforces a minimum total test coverage threshold.
- `make changelog` writes `CHANGELOG.md` in the repo using `git-cliff`.
- `make release` runs CI checks, calculates the next version with `git-cliff`, updates `VERSION` and `CHANGELOG.md`, creates a dedicated `chore(release): vX.Y.Z` commit, and tags that commit locally.
- `make release` does not push or publish anything.
- `make release` requires `git-cliff` to be installed locally.
