#!/usr/bin/env bash
# Verifies the project has its own Git repository with a first commit (not the
# enclosing startups repository). Exit codes: 4 wrong/no repository, 5 no commit.
set -u
root="$(cd "$(dirname "$0")/.." && pwd -P)"
top="$(git -C "$root" rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$top" ] || [ "$(cd "$top" && pwd -P)" != "$root" ]; then
  echo "GIT CHECK FAILED: project needs its own Git repository: $root"
  echo "Detected Git root: ${top:-none}. Do not use the parent repository."
  echo "User action: cd '$root' && git init -b main && git add . && git commit -m 'Prepare development workspace'"
  exit 4
fi
if ! git -C "$root" rev-parse --verify -q HEAD >/dev/null; then
  echo "GIT CHECK FAILED: create the first commit before development."
  exit 5
fi
echo "GIT CHECK OK: $root"
