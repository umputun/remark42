#!/bin/sh
# Exercise the stamp without building an image or starting the browser stack.
set -eu

stamp_script=$(cd "$(dirname "$0")" && pwd)/stamp.sh
work=$(mktemp -d "${TMPDIR:-/tmp}/remark42-stamp.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
mkdir -p "$work/e2e/telegramstub" "$work/backend" "$work/frontend"
cp "$stamp_script" "$work/e2e/stamp.sh"
cd "$work"
git init -q
git config user.name w3lld1
git config user.email w3lld1@users.noreply.github.com
printf 'original\n' > backend/source
printf 'frontend\n' > 'frontend/file with spaces'
printf 'stub\n' > e2e/telegramstub/source
printf 'image\n' > Dockerfile
printf 'init\n' > docker-init.sh
git add .
git commit -qm initial

stamp() { E2E_COVERAGE='' sh e2e/stamp.sh; }
same() {
	if [ "$1" != "$2" ]; then
		printf 'FAIL: %s (%s != %s)\n' "$3" "$1" "$2" >&2
		exit 1
	fi
	printf 'PASS: %s\n' "$3"
}
different() {
	if [ "$1" = "$2" ]; then
		printf 'FAIL: %s (%s == %s)\n' "$3" "$1" "$2" >&2
		exit 1
	fi
	printf 'PASS: %s\n' "$3"
}

original=$(stamp)
printf 'edited\n' > backend/source
edited=$(stamp)
different "$original" "$edited" 'tracked content changes the stamp'
git add backend/source
same "$edited" "$(stamp)" 'staging an edit preserves the stamp'
git commit -qm edit
same "$edited" "$(stamp)" 'committing an edit preserves the stamp'

rm backend/source
deleted=$(stamp)
different "$edited" "$deleted" 'deleting a tracked file changes the stamp'
git add backend/source
same "$deleted" "$(stamp)" 'staging a deletion preserves the stamp'
git commit -qm deletion
same "$deleted" "$(stamp)" 'committing a deletion preserves the stamp'

printf 'changed\n' > 'frontend/file with spaces'
spaced=$(stamp)
different "$deleted" "$spaced" 'content in a path with spaces is hashed'
git add frontend
git commit -qm spaces
same "$spaced" "$(stamp)" 'committing a path with spaces preserves the stamp'

printf 'new\n' > frontend/untracked
untracked=$(stamp)
different "$spaced" "$untracked" 'untracked names change the stamp'
printf 'later edit\n' > frontend/untracked
same "$untracked" "$(stamp)" 'untracked contents retain the name-only behavior'
mv frontend/untracked frontend/renamed
different "$untracked" "$(stamp)" 'renaming an untracked file changes the stamp'
rm frontend/renamed
same "$spaced" "$(stamp)" 'removing an untracked file restores the stamp'

printf 'suite only\n' > e2e/test.txt
git add e2e/test.txt
git commit -qm suite
same "$spaced" "$(stamp)" 'a suite-only commit preserves the stamp'
printf 'edited stub\n' > e2e/telegramstub/source
different "$spaced" "$(stamp)" 'the telegram image sources are included'
plain=$(stamp)
same "$plain" "$(E2E_COVERAGE=0 sh e2e/stamp.sh)" 'coverage=0 is a plain build'
different "$plain" "$(E2E_COVERAGE=1 sh e2e/stamp.sh)" 'coverage=1 is a separate build'
