#!/bin/sh
# Digest of everything that ends up in an image the stack runs.
#
# The compose stack tags its image ghcr.io/umputun/remark42:dev, which every checkout of this
# repository shares, so a stack brought up from one worktree answers on the same ports as one
# brought up from another. The suite stamps the image it builds with this value and refuses a
# running stack carrying a different one, so it never tests code nobody is looking at.
#
# e2e/telegramstub is in the list because compose builds it too: it is a second image made from
# sources in this repository, and leaving it out let an edited stub run behind a stack the guard
# had just accepted.
#
# Tracked content is covered exactly; an untracked file changes the digest when it appears,
# by name, but later edits to it do not.
set -eu

cd "$(dirname "$0")/.."

sources="backend frontend Dockerfile docker-init.sh e2e/telegramstub"

digest() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum
	else
		shasum -a 256
	fi
}

{
	# Hash the files on disk, not HEAD plus a diff: staging or committing the same
	# content must not invalidate a running stack. Skip deleted files, including
	# staged deletions, so committing their removal leaves the stamp unchanged too.
	# shellcheck disable=SC2086,SC2016 # split paths; expand $file in the child shell
	git ls-files -z -- $sources | xargs -0 sh -c '
		for file do
			if [ -f "$file" ]; then
				if command -v sha256sum >/dev/null 2>&1; then
					sha256sum "$file"
				else
					shasum -a 256 "$file"
				fi
			fi
		done
	' sh
	# Only untracked names belong here; porcelain also records index state.
	# shellcheck disable=SC2086
	git ls-files --others --exclude-standard -z -- $sources
	# an instrumented build is a different binary from the same sources, so it has to be a
	# different stamp: without this a coverage stack is accepted for a plain run, and a plain
	# stack for a coverage run, which reports no coverage at all and looks like untested code
	# only "1" instruments, which is the Dockerfile's test too: anything else has to digest as
	# the plain build, or E2E_COVERAGE=0 becomes a third stamp for a stack built the ordinary way
	case "${E2E_COVERAGE:-}" in
	1) printf 'coverage=1\n' ;;
	*) printf 'coverage=\n' ;;
	esac
} | digest | cut -c1-16
