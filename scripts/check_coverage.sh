#!/usr/bin/env bash

set -euo pipefail

threshold="${1:-75}"
profile="coverage.out"

go test ./... -covermode=atomic -coverpkg=./... -coverprofile="${profile}" >/dev/null

total="$(go tool cover -func="${profile}" | awk '/^total:/ {print substr($3, 1, length($3)-1)}')"

awk -v total="${total}" -v threshold="${threshold}" 'BEGIN {
  if (total + 0 < threshold + 0) {
    printf("coverage %.1f%% is below threshold %.1f%%\n", total, threshold)
    exit 1
  }
  printf("coverage %.1f%% meets threshold %.1f%%\n", total, threshold)
}'
