# workflow-lock

`workflow-lock` keeps workflow files readable while locking remote action refs to immutable commit SHAs in `workflow-lock.yaml`.

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
make verify
make build
make dist
make clean
```

## Releases

- Push a tag like `v0.1.0` to build release artifacts and attach them to a GitHub release.
- `make dist` builds local cross-platform binaries and writes `dist/SHA256SUMS`.
