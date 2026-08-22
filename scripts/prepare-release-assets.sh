#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
APP_DIR="$ROOT/frontend/apps/remark42"
PUBLIC_DIR="$APP_DIR/public"
EMBED_DIR="$ROOT/backend/app/cmd/web"
PREPARED_MARKER="$EMBED_DIR/.release-assets-prepared"

for cmd in git pnpm; do
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
  cd "$APP_DIR"
  if [[ "${SKIP_PNPM_INSTALL:-}" != "true" ]]; then
    CI=true pnpm install --frozen-lockfile
  fi
  CI=true pnpm build
)

cp -R "$PUBLIC_DIR"/. "$EMBED_DIR"/

# the placeholder stays in: the server fills it with its own REMARK_URL when it serves these files,
# which is the only chance the binary gets. substituting a value here would bake one instance's
# address into every copy of the release
if ! grep -Rq "{% REMARK_URL %}" "$EMBED_DIR"; then
  echo "error: no REMARK_URL placeholder in $EMBED_DIR, the build stopped templating it" >&2
  exit 1
fi

touch "$PREPARED_MARKER"
