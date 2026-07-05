OS=linux
ARCH=amd64
GITHUB_REF=$(shell git rev-parse --symbolic-full-name HEAD)
GITHUB_SHA=$(shell git rev-parse --short HEAD)
CLEANUP_RELEASE_ASSETS=$(CURDIR)/scripts/cleanup-release-assets.sh

bin:
	@set -e; \
		./scripts/prepare-release-assets.sh; \
		trap '$(CLEANUP_RELEASE_ASSETS)' EXIT; \
		cd backend && CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -o ../remark42 -ldflags "-X main.revision=$(GITHUB_REF)-$(GITHUB_SHA) -s -w" ./app

docker:
	DOCKER_BUILDKIT=1 docker build -t umputun/remark42 -t ghcr.io/umputun/remark42 --build-arg GITHUB_REF=$(GITHUB_REF) --build-arg GITHUB_SHA=$(GITHUB_SHA) \
		--build-arg CI=true --build-arg SKIP_FRONTEND_TEST=true --build-arg SKIP_BACKEND_TEST=true .

dockerx:
	docker buildx build --build-arg GITHUB_REF=$(GITHUB_REF) --build-arg GITHUB_SHA=$(GITHUB_SHA) --build-arg CI=true \
		--build-arg SKIP_FRONTEND_TEST=true --build-arg SKIP_BACKEND_TEST=true \
		--progress=plain --platform linux/amd64,linux/arm64 \
		-t ghcr.io/umputun/remark42:master -t umputun/remark42:master .

release:
	@set -e; \
		trap '$(CLEANUP_RELEASE_ASSETS)' EXIT; \
		goreleaser release --snapshot --clean --skip=publish

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

.PHONY: bin docker dockerx release race_test backend frontend rundev e2e
