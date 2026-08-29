#!/bin/sh
# Digest of everything that ends up in the e2e image.
#
# The compose stack tags its image ghcr.io/umputun/remark42:dev, which every checkout of this
# repository shares, so a stack brought up from one worktree answers on the same ports as one
# brought up from another. The suite stamps the image it builds with this value and refuses a
# running stack carrying a different one, so it never tests code nobody is looking at.
#
# Tracked content is covered exactly; an untracked file changes the digest when it appears,
# by name, but later edits to it do not.
set -eu

cd "$(dirname "$0")/.."

sources="backend frontend Dockerfile docker-init.sh"

digest() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum
	else
		shasum -a 256
	fi
}

{
	# the content of those paths, not the commit: an e2e-only commit cannot change the image, and
	# keying on HEAD would rebuild the stack for every one of them
	# shellcheck disable=SC2086 # the path list is deliberately split into arguments
	for path in $sources; do
		git rev-parse "HEAD:$path"
	done
	# shellcheck disable=SC2086
	git diff HEAD -- $sources
	# shellcheck disable=SC2086
	git status --porcelain -- $sources
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
