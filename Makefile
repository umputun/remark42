OS=linux
ARCH=amd64
GITHUB_REF=$(shell git rev-parse --symbolic-full-name HEAD)
GITHUB_SHA=$(shell git rev-parse --short HEAD)

bin:
	docker build -f Dockerfile.artifacts -t remark42.bin .
	- @docker rm -f remark42.bin 2>/dev/null || exit 0
	docker run -d --name=remark42.bin remark42.bin
	docker cp remark42.bin:/artifacts/remark42.$(OS)-$(ARCH) remark42
	docker rm -f remark42.bin

docker:
	DOCKER_BUILDKIT=1 docker build -t umputun/remark42 --build-arg GITHUB_REF=$(GITHUB_REF) --build-arg GITHUB_SHA=$(GITHUB_SHA) \
		--build-arg CI=true --build-arg SKIP_FRONTEND_TEST=true --build-arg SKIP_BACKEND_TEST=true .

dockerx:
	docker buildx build --build-arg GITHUB_REF=$(GITHUB_REF) --build-arg GITHUB_SHA=$(GITHUB_SHA) --build-arg CI=true \
		--build-arg SKIP_FRONTEND_TEST=true --build-arg SKIP_BACKEND_TEST=true \
		--progress=plain --platform linux/amd64,linux/arm/v7,linux/arm64 \
		-t ghcr.io/umputun/remark42:master -t umputun/remark42:master .

release:
	docker build -f Dockerfile.artifacts --no-cache --pull --build-arg CI=true \
		--build-arg GITHUB_REF=$(GITHUB_REF) --build-arg GITHUB_SHA=$(GITHUB_SHA) -t remark42.bin .
	- @docker rm -f remark42.bin 2>/dev/null || exit 0
	- @mkdir -p bin
	docker run -d --name=remark42.bin remark42.bin
	docker cp remark42.bin:/artifacts/remark42.linux-amd64.tar.gz bin/remark42.linux-amd64.tar.gz
	docker cp remark42.bin:/artifacts/remark42.linux-386.tar.gz bin/remark42.linux-386.tar.gz
	docker cp remark42.bin:/artifacts/remark42.linux-arm64.tar.gz bin/remark42.linux-arm64.tar.gz
	docker cp remark42.bin:/artifacts/remark42.darwin-amd64.tar.gz bin/remark42.darwin-amd64.tar.gz
	docker cp remark42.bin:/artifacts/remark42.darwin-arm64.tar.gz bin/remark42.darwin-arm64.tar.gz
	docker cp remark42.bin:/artifacts/remark42.freebsd-amd64.tar.gz bin/remark42.freebsd-amd64.tar.gz
	docker cp remark42.bin:/artifacts/remark42.windows-amd64.zip bin/remark42.windows-amd64.zip
	docker rm -f remark42.bin

race_test:
	cd backend/app && go test -race -timeout=60s -count 1 ./...

backend:
	docker compose -f compose-dev-backend.yml build

frontend:
	docker compose -f compose-dev-frontend.yml build

rundev:
	SKIP_BACKEND_TEST=true SKIP_FRONTEND_TEST=true GITHUB_REF=$(GITHUB_REF) GITHUB_SHA=$(GITHUB_SHA) CI=true \
		docker compose -f compose-private.yml build
	docker compose -f compose-private.yml up

e2e:
	docker compose -f compose-e2e-test.yml up --build --quiet-pull --exit-code-from tests

.PHONY: bin backend
