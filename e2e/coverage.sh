#!/bin/sh
# Statement coverage of the backend as the browser suite exercises it.
#
# The code under test runs in a container instead of in the test process, so `go test -cover`
# cannot see it. The image is built with `go build -cover` instead, each instance writes a profile
# to a mounted directory as it exits, and the profiles are merged here.
#
# The instrumented services are read out of the coverage overlay, so that file is the only place
# the list is written down.
set -eu

cd "$(dirname "$0")/.."

compose="-f compose-e2e-test.yml -f compose-e2e-coverage.yml"
instances=$(awk '/^  [a-z0-9-]*:$/ { gsub(/[ :]/, ""); print }' compose-e2e-coverage.yml)

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

# the profiles are on the host, so the stack has nothing left to hold, and leaving it half stopped
# is what the next plain run would find
# shellcheck disable=SC2086
docker compose $compose down -v >/dev/null 2>&1

exit $status
