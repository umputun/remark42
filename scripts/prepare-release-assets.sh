#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
FRONTEND_DIR="$ROOT/frontend"
APP_DIR="$FRONTEND_DIR/apps/remark42"
PUBLIC_DIR="$APP_DIR/public"
EMBED_DIR="$ROOT/backend/app/cmd/web"
PREPARED_MARKER="$EMBED_DIR/.release-assets-prepared"

for cmd in git pnpm perl; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: $cmd is required to build release assets" >&2
    exit 1
  fi
done

if [[ -n $(git -C "$ROOT" status --porcelain -- backend/app/cmd/web) ]]; then
  echo "error: backend/app/cmd/web has uncommitted changes" >&2
  exit 1
fi

cleanup_on_error() {
  git -C "$ROOT" checkout -- backend/app/cmd/web
  git -C "$ROOT" clean -fdX backend/app/cmd/web frontend/apps/remark42/public >/dev/null
}

cleanup_on_exit() {
  rc=$?
  if [[ "$rc" -ne 0 ]]; then
    cleanup_on_error
  fi
}

trap cleanup_on_exit EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

rm -rf "$PUBLIC_DIR" "$EMBED_DIR"
mkdir -p "$EMBED_DIR"

(
  cd "$FRONTEND_DIR"
  if [[ "${SKIP_PNPM_INSTALL:-}" != "true" ]]; then
    CI=true pnpm install --frozen-lockfile
  fi
  cd "$APP_DIR"
  CI=true pnpm build
)

cp -R "$PUBLIC_DIR"/. "$EMBED_DIR"/

find "$EMBED_DIR" -type f \( -name '*.html' -o -name '*.js' -o -name '*.mjs' \) \
  -exec perl -pi -e 's|\{\% REMARK_URL \%\}|http://127.0.0.1:8080|g' {} +

if grep -R "{% REMARK_URL %}" "$EMBED_DIR" >/dev/null; then
  echo "error: unreplaced REMARK_URL placeholder in $EMBED_DIR" >&2
  exit 1
fi

touch "$PREPARED_MARKER"
