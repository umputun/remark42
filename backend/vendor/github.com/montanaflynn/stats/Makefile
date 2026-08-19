.PHONY: all

default: test lint

format: 
	go fmt .

test:
	go test -race 
	
check: format test

benchmark:
	go test -bench=. -benchmem

coverage:
	go test -coverprofile=coverage.out
	go tool cover -html="coverage.out"

lint: format
	golangci-lint run .

docs:
	godoc2md github.com/montanaflynn/stats | sed -e s#src/target/##g > DOCUMENTATION.md

release:
	@test -n "${TAG}" || { echo "TAG is required, e.g. make release TAG=v0.10.0"; exit 1; }
	sh scripts/update-changelog.sh ${TAG}
	git add CHANGELOG.md
	git commit -m "chore: update changelog for ${TAG}"
	git tag -a ${TAG} -m "${TAG}"
	git push origin master ${TAG}

