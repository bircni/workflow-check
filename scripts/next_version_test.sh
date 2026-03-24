#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="${root}/scripts/next_version.sh"

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

cd "${tmpdir}"
git init -q
git config user.name "Test"
git config user.email "test@example.com"

touch file.txt
git add file.txt
git commit -q -m "chore: initial"
git tag -a v1.2.3 -m v1.2.3

if "${script}" "v1.2.3" >/dev/null 2>&1; then
  echo "expected next_version to fail when there are no new commits" >&2
  exit 1
fi

echo "a" >> file.txt
git commit -qam "fix: patch release"
[[ "$("${script}" "v1.2.3")" == "v1.2.4" ]]

echo "b" >> file.txt
git commit -qam "feat: minor release"
[[ "$("${script}" "v1.2.3")" == "v1.3.0" ]]

echo "c" >> file.txt
git commit -qam "feat!: major release"
[[ "$("${script}" "v1.2.3")" == "v2.0.0" ]]

echo "next_version tests passed"
