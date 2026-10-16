#!/usr/bin/env bash

# Local pre-push equivalent of CI. It deliberately validates only the current
# checked-out HEAD and refuses to touch a pre-existing test fixture.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"
export TZ=UTC

die() {
  printf 'CI pre-push gate: %s\n' "$*" >&2
  exit 1
}

require_clean_tracked_tree() {
  git diff --quiet || die 'tracked worktree has unstaged changes; commit or stash them before pushing'
  git diff --cached --quiet || die 'tracked index has staged changes; commit or unstage them before pushing'
}

bash tools/check-git.sh
require_clean_tracked_tree
checked_head="$(git rev-parse HEAD)"

# Lefthook forwards Git's pre-push stdin. A deletion has no source tree to
# validate; every non-deletion ref must publish the exact checked-out commit.
while IFS=' ' read -r local_ref local_oid remote_ref remote_oid extra; do
  if [[ -z "$local_ref" && -z "$local_oid" && -z "$remote_ref" && -z "$remote_oid" ]]; then
    continue
  fi
  [[ -z "$extra" && -n "$local_ref" && -n "$local_oid" && -n "$remote_ref" && -n "$remote_oid" ]] || \
    die 'received malformed pre-push input'
  if [[ "$local_oid" =~ ^0+$ ]]; then
    continue
  fi
  [[ "$local_oid" == "$checked_head" ]] || \
    die "push ref $local_ref points to $local_oid, not checked-out HEAD $checked_head; checkout that commit before pushing"
done

node_bin="$root/docs/justix-auto/dev/local/toolchains/node-24.21.0/bin"
if [[ -x "$node_bin/node" && -x "$node_bin/npm" ]]; then
  export PATH="$node_bin:$PATH"
fi

command -v node >/dev/null 2>&1 || die 'Node.js is required (expected 24.21.0)'
command -v npm >/dev/null 2>&1 || die 'npm is required (expected 11.19.0)'
[[ "$(node --version)" == 'v24.21.0' ]] || die "expected Node.js v24.21.0, found $(node --version)"
[[ "$(npm --version)" == '11.19.0' ]] || die "expected npm 11.19.0, found $(npm --version)"

git ls-files -z -- ':(glob)**/*.go' | while IFS= read -r -d '' go_file; do
  case "$go_file" in
    cmd/*.go|internal/*.go|migrations/*.go) ;;
    *) die "tracked Go source outside local CI roots: $go_file" ;;
  esac
done

pkg='./cmd/... ./internal/... ./migrations/...'
make --no-print-directory -j1 web-install
make --no-print-directory -j1 lint typecheck test-web PKG="$pkg"

docker_endpoint="$(docker --context desktop-linux context inspect desktop-linux --format '{{(index .Endpoints "docker").Host}}')" || \
  die 'cannot inspect Docker context desktop-linux'
[[ "$docker_endpoint" == unix://* ]] || \
  die "Docker context desktop-linux must use a local unix endpoint; found $docker_endpoint"

fixture_label='com.docker.compose.project=justixauto-test'
containers="$(docker --context desktop-linux ps -aq --filter "label=$fixture_label")"
volumes="$(docker --context desktop-linux volume ls -q --filter "label=$fixture_label")"
networks="$(docker --context desktop-linux network ls -q --filter "label=$fixture_label")"
if [[ -n "$containers" || -n "$volumes" || -n "$networks" ]]; then
  die 'refusing to run Go tests: Docker project justixauto-test already has containers, volumes, or networks; preserve and remove that fixture yourself before retrying'
fi

make --no-print-directory -j1 test-go \
  PKG="$pkg" \
  TEST_COMPOSE='docker --context desktop-linux compose --project-name justixauto-test -f deploy/test/compose.yaml'

require_clean_tracked_tree
[[ "$(git rev-parse HEAD)" == "$checked_head" ]] || \
  die "HEAD changed during CI pre-push gate (started at $checked_head); rerun from the checked-out commit"
printf 'CI pre-push gate passed for %s\n' "$checked_head"
