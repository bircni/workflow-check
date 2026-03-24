#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

if [[ -n "$(git status --short)" ]]; then
  echo "working tree is not clean" >&2
  exit 1
fi

if ! command -v git-cliff >/dev/null 2>&1; then
  echo "git-cliff is required; install it first (for example: cargo install git-cliff --locked)" >&2
  exit 1
fi

latest_tag="$(git tag --sort=version:refname | tail -n1)"
if [[ -n "${latest_tag}" ]] && [[ "$(git rev-list --count "${latest_tag}..HEAD")" -eq 0 ]]; then
  echo "no commits since ${latest_tag}" >&2
  exit 1
fi

make ci

next_tag="$(git-cliff --config .git-cliff.toml --bumped-version | tail -n1)"

if [[ -f CHANGELOG.md ]]; then
  git-cliff --config .git-cliff.toml --unreleased --tag "${next_tag}" --prepend CHANGELOG.md
else
  git-cliff --config .git-cliff.toml --output CHANGELOG.md
fi

git add CHANGELOG.md
git commit -m "chore(release): ${next_tag}"
git tag -a "${next_tag}" -m "${next_tag}"
printf 'created release commit and tag %s\n' "${next_tag}"
