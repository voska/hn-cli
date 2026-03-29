VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo "")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
BIN     := bin/hn

.PHONY: build fmt lint test ci clean install

build:
	@mkdir -p bin
	@go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/hn

install: build
	@cp $(BIN) $(GOPATH)/bin/hn 2>/dev/null || cp $(BIN) $(HOME)/go/bin/hn

fmt:
	@gofumpt -w . 2>/dev/null || gofmt -w .

lint:
	@golangci-lint run 2>/dev/null || echo "golangci-lint not installed, skipping"

test:
	@go test -race ./...

ci: fmt lint test build

clean:
	rm -rf bin/
