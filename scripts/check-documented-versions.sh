#!/usr/bin/env bash
# the installation page states the toolchain a source build needs. nothing else keeps those
# numbers in step with the manifests they describe, and the drift is silent: a wrong version in
# the docs builds and tests exactly as well as a right one.
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
DOC="$ROOT/site/content/docs/getting-started/installation/index.md"
GOMOD="$ROOT/backend/go.mod"
PKG="$ROOT/frontend/apps/remark42/package.json"
NVMRC="$ROOT/frontend/.nvmrc"

fail=0

# every extraction is checked rather than allowed to come back empty, since an empty expectation
# compares equal to nothing and the check would pass by doing nothing
need() {
  local what="$1" value="$2" source="$3"
  if [[ -z "$value" ]]; then
    echo "error: could not read $what from $source" >&2
    exit 1
  fi
}

go_want=$(sed -n 's/^go \([0-9]*\.[0-9]*\).*/\1/p' "$GOMOD" | head -1)
need "the go version" "$go_want" "$GOMOD"

node_want=$(sed -n 's/.*"node": *">=\([0-9]*\)".*/\1/p' "$PKG" | head -1)
need "engines.node" "$node_want" "$PKG"

pnpm_want=$(sed -n 's/.*"packageManager": *"pnpm@\([0-9]*\)\..*/\1/p' "$PKG" | head -1)
need "the packageManager major" "$pnpm_want" "$PKG"

nvmrc=$(tr -d '[:space:]' < "$NVMRC")
need "the node version" "$nvmrc" "$NVMRC"

check() {
  local label="$1" pattern="$2" want="$3"
  local found
  found=$(grep -oE "$pattern" "$DOC" | sort -u || true)
  if [[ -z "$found" ]]; then
    echo "error: no $label version stated in ${DOC#"$ROOT"/}, so this check has nothing to compare" >&2
    fail=1
    return
  fi
  local line
  while read -r line; do
    if [[ "$line" != "$want" ]]; then
      echo "error: ${DOC#"$ROOT"/} says '$line' where the repository says '$want'" >&2
      fail=1
    fi
  done <<< "$found"
}

check "Go" "Go [0-9]+\.[0-9]+" "Go $go_want"
check "Node" "Node [0-9]+\+" "Node $node_want+"
check "PNPM" "PNPM [0-9]+" "PNPM $pnpm_want"

if [[ "$nvmrc" != "$node_want" ]]; then
  echo "error: frontend/.nvmrc says '$nvmrc' where engines.node says '>=$node_want'" >&2
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi
echo "documented versions match the repository: Go $go_want, Node $node_want+, PNPM $pnpm_want"
