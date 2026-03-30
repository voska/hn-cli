VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo "")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
BIN     := bin/hn
GOPATH  := $(shell go env GOPATH)

.PHONY: build fmt lint test ci clean install release

build:
	@mkdir -p bin
	@go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/hn

install: build
	@install -d "$(GOPATH)/bin"
	@install $(BIN) "$(GOPATH)/bin/hn"

fmt:
	@gofumpt -w . 2>/dev/null || gofmt -w .

lint:
	golangci-lint run ./...

test:
	@go test -race ./...

ci: fmt lint test build

release:
	goreleaser release --clean

clean:
	rm -rf bin/ dist/
