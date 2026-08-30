#!/bin/sh
# What the browser suite reaches, in the backend and in the widget.
#
# Neither half is visible to the tool that usually measures it. The backend runs in a container
# instead of in the test process, so `go test -cover` sees nothing; the widget runs in the
# browser, and jest reports only what its own suites reach. Both are therefore instrumented at
# build time: the backend with `go build -cover`, which writes a profile as each instance exits,
# and the bundle with babel-plugin-istanbul, which counts on the page for the suite to read.
#
# The instrumented services are read out of the coverage overlay, so that file is the only place
# the list is written down.
set -eu

cd "$(dirname "$0")/.."

compose="-f compose-e2e-test.yml -f compose-e2e-coverage.yml"
instances=$(awk '/^  [a-z0-9-]*:$/ { gsub(/[ :]/, ""); print }' compose-e2e-coverage.yml)

# checked before the build rather than after the suite, since the widget half cannot be reported
# without it and the run costs several minutes either way
if [ ! -d frontend/apps/remark42/node_modules ]; then
	echo "frontend/apps/remark42/node_modules is missing, so the widget counters could not be reported"
	echo "run pnpm install there first"
	exit 1
fi

./e2e/tls/generate.sh

# an interrupted run leaves the stack up, and `up` below would not recreate a container whose
# configuration has not changed: it would keep the mount it already holds, which the wipe just
# unlinked, and write its profile into a directory nothing can read. Start from no stack at all.
# shellcheck disable=SC2086
docker compose $compose down -v >/dev/null 2>&1 || true

rm -rf e2e/coverage
# the container writes as its own user, and this directory is generated, gitignored, and holds
# nothing but profiles
for i in $instances; do
	install -d -m 0777 "e2e/coverage/$i"
done

# the stack comes down however this ends. Everything below can exit under `set -e`, and a running
# or half-stopped stack is what the next plain run would then find
# shellcheck disable=SC2064 # $compose is meant to expand now, while it is still in scope
trap "docker compose $compose down -v >/dev/null 2>&1 || true" EXIT

# shellcheck disable=SC2086 # the compose file list is deliberately split into arguments
E2E_COVERAGE=1 E2E_STAMP=$(E2E_COVERAGE=1 ./e2e/stamp.sh) \
	docker compose $compose up -d --build --quiet-pull --wait

# the suite's exit status is kept, but the instances are stopped either way. A profile is written
# as the process exits, so returning early on a failing run would discard the coverage of the run
# that most needed explaining, and leave the stack up as well.
status=0
E2E_COVERAGE=1 make e2e || status=$?

# shellcheck disable=SC2086
docker compose $compose stop -t 30 $instances

# every instance is checked, not the total: one killed while the others exited cleanly would
# otherwise merge into a plausible report with its own code reading as unreached
missing=""
for i in $instances; do
	if [ -z "$(find "e2e/coverage/$i" -name 'covcounters.*' 2>/dev/null)" ]; then
		missing="$missing $i"
	fi
done

if [ -n "$missing" ]; then
	echo "no coverage profile from:$missing"
	echo "those instances did not shut down cleanly; the report would understate them silently"
	status=1
else
	dirs=""
	for i in $instances; do
		dirs="${dirs:+$dirs,}e2e/coverage/$i"
	done
	go tool covdata textfmt -i="$dirs" -o e2e/coverage/e2e.cov_tmp
	# the same exclusion the unit profile applies, so the two can be added together
	grep -v "_mock.go" e2e/coverage/e2e.cov_tmp >e2e/coverage/e2e.cov
	rm e2e/coverage/e2e.cov_tmp
	(cd backend && go tool cover -func=../e2e/coverage/e2e.cov | tail -1)
fi

# The widget half. The bundle counts what runs and the suite reads the counters out of each page
# as it closes, so an empty directory means the pages were never asked, not that nothing ran.
if [ -z "$(find e2e/coverage/frontend -name '*.json' 2>/dev/null)" ]; then
	echo "no widget coverage was collected: the bundle was not instrumented, or no page was read"
	status=1
else
	# nyc merges the directory itself, and it runs from the package so the rewritten paths resolve.
	# its failure folds into the status the same way the suite's does, since errexit would
	# otherwise end the run here and say nothing about the half that did work
	(
		cd frontend/apps/remark42 &&
			pnpm exec nyc report \
				--temp-dir ../../../e2e/coverage/frontend \
				--report-dir ../../../e2e/coverage/frontend-report \
				--reporter=lcov --reporter=text-summary
	) || status=$?
fi

exit $status
