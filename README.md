# workflow-lock

`workflow-lock` keeps workflow files readable while locking remote action refs to immutable commit SHAs in `workflow-lock.yaml`.

The current tool version is tracked in [VERSION](VERSION) and exposed by `workflow-lock version`.

It supports GitHub-style and Gitea-style refs, including:

- `owner/repo@ref`
- `owner/repo/.github/workflows/file.yml@ref`
- `github.com/owner/repo@ref`
- `gitea.example.com/owner/repo@ref`

Ignored in v1:

- local `./...` and `../...` refs
- generic URL forms such as `https://...`

## Installation

**Prebuilt binaries** for common platforms are attached to [GitHub Releases](https://github.com/bircni/workflow-check/releases). Verify with the published `SHA256SUMS` when you download.

**Go toolchain** (installs `workflow-lock` into `GOBIN` or `GOPATH/bin`; ensure that directory is on your `PATH`):

```bash
go install github.com/bircni/workflow-check/cmd/workflow-lock@latest
```

## Usage

After installation, run the `workflow-lock` binary (or `./bin/workflow-lock` if you used `make build`):

```bash
workflow-lock lock
workflow-lock verify
workflow-lock list
workflow-lock diff
workflow-lock version
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
workflow-lock lock -default-host code.gitea.example.com
workflow-lock diff -format json
workflow-lock verify -config .workflow-lock-config.yaml
```

When hacking on this repository without installing, you can use `go run ./cmd/workflow-lock <subcommand> ...` instead of `workflow-lock`.

## Config

If `.workflow-lock-config.yaml` exists in the repo root, the CLI loads it automatically. Flags still override config values.

Example:

```yaml
workflows: .github/workflows
lockfile: workflow-lock.yaml
default_host: github.com
```

A ready-to-copy example lives at [.workflow-lock-config.example.yaml](.workflow-lock-config.example.yaml).

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to propose changes.

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
