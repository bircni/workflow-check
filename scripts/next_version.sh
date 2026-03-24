#!/usr/bin/env bash

set -euo pipefail

latest_tag="${1:-}"

if [[ -z "${latest_tag}" ]]; then
  latest_tag="$(git tag --sort=version:refname | tail -n1)"
fi

if [[ -z "${latest_tag}" ]]; then
  latest_tag="v0.0.0"
  range="HEAD"
else
  range="${latest_tag}..HEAD"
  if [[ "$(git rev-list --count "${range}")" -eq 0 ]]; then
    echo "no commits since ${latest_tag}" >&2
    exit 1
  fi
fi

version="${latest_tag#v}"
IFS='.' read -r major minor patch <<< "${version}"

bump="patch"
while IFS= read -r subject; do
  [[ -z "${subject}" ]] && continue
  if printf '%s\n' "${subject}" | grep -Eq '^[a-z]+(\([^)]*\))?!:'; then
    bump="major"
    break
  fi
done < <(git log --format=%s "${range}")

if [[ "${bump}" != "major" ]]; then
  while IFS= read -r body; do
    if [[ "${body}" == *"BREAKING CHANGE:"* ]]; then
      bump="major"
      break
    fi
  done < <(git log --format=%b "${range}")
fi

if [[ "${bump}" == "patch" ]]; then
  while IFS= read -r subject; do
    if printf '%s\n' "${subject}" | grep -Eq '^feat(\([^)]*\))?:'; then
      bump="minor"
      break
    fi
  done < <(git log --format=%s "${range}")
fi

case "${bump}" in
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  patch)
    patch=$((patch + 1))
    ;;
  *)
    echo "unsupported bump type: ${bump}" >&2
    exit 1
    ;;
esac

printf 'v%s.%s.%s\n' "${major}" "${minor}" "${patch}"
