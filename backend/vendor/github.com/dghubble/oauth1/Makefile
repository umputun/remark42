.PHONY: all
all: test vet lint fmt

.PHONY: test
test:
	@go test $$(go list ./... | grep -v examples) -cover

.PHONY: vet
vet:
	@go vet -all $$(go list ./... | grep -v examples)

.PHONY: lint
lint:
	@golint -set_exit_status ./...

.PHONY: fmt
fmt:
	@test -z $$(go fmt ./...)

