BINARY=lightctl
CMD=./cmd/lightctl
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -ldflags "-X github.com/aviadshiber/lightctl/cmd.Version=$(VERSION)"

.PHONY: build install test lint release

build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

install:
	go install $(LDFLAGS) $(CMD)

test:
	go test ./... -race

lint:
	golangci-lint run

release:
	goreleaser release --clean
