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

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required to publish releases" >&2
  exit 1
fi

latest_tag="$(git tag --sort=version:refname | tail -n1)"
next_tag="$(./scripts/next_version.sh "${latest_tag}")"

if [[ "${latest_tag}" == "${next_tag}" ]]; then
  echo "refusing to create duplicate release ${next_tag}" >&2
  exit 1
fi

make ci

notes_file="$(mktemp)"
trap 'rm -f "${notes_file}"' EXIT

if [[ -n "${latest_tag}" ]]; then
  git-cliff --config .git-cliff.toml --latest --strip header > "${notes_file}"
else
  git-cliff --config .git-cliff.toml --unreleased --strip header > "${notes_file}"
fi

git tag -a "${next_tag}" -m "${next_tag}"
git push origin main
git push origin "${next_tag}"
gh release create "${next_tag}" --notes-file "${notes_file}" --title "${next_tag}"
