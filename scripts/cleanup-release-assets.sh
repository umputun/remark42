#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
PREPARED_MARKER="$ROOT/backend/app/cmd/web/.release-assets-prepared"

if [[ ! -e "$PREPARED_MARKER" ]]; then
  exit 0
fi

git -C "$ROOT" checkout -- backend/app/cmd/web
git -C "$ROOT" clean -fdX backend/app/cmd/web frontend/apps/remark42/public >/dev/null
