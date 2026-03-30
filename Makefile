BINARY=lightctl
CMD=./cmd/lightctl

.PHONY: build install test lint release

build:
	go build -o $(BINARY) $(CMD)

install:
	go install $(CMD)

test:
	go test ./... -race

lint:
	golangci-lint run

release:
	goreleaser release --clean
