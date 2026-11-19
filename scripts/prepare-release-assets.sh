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

# `//go:embed web` in app/cmd/server.go takes the default pattern rules, which skip every name
# beginning with a dot or an underscore. A bundle emitted under such a name would be absent from
# the binary while still present in frontend/apps/remark42/public, so it would serve correctly from
# --web-root and 404 from a released binary, which is the arrangement nobody tests. Nothing emits
# one today; this fails the release if that ever changes, instead of shipping the hole.
# The marker below is written after this check because it is itself such a name.
# sed, not head: head closes the pipe once it has its five lines, and on a tree with enough of
# them sort dies of SIGPIPE, which pipefail turns into a failed assignment and errexit turns into
# an exit before the message below is ever printed
hidden=$(find "$EMBED_DIR" -mindepth 1 \( -name '.*' -o -name '_*' \) | sort | sed -n '1,5p')
if [[ -n "$hidden" ]]; then
  echo "error: go:embed would drop these from the binary, since it ignores names starting with a" >&2
  echo "dot or an underscore, and they would 404 from a release while working under --web-root:" >&2
  echo "$hidden" >&2
  exit 1
fi

touch "$PREPARED_MARKER"
