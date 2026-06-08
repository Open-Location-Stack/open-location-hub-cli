BINARY ?= olh
DIST ?= dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -s -w -X github.com/formation-res/open-location-hub-cli/internal/build.Version=$(VERSION) -X github.com/formation-res/open-location-hub-cli/internal/build.Commit=$(COMMIT) -X github.com/formation-res/open-location-hub-cli/internal/build.Date=$(DATE)

.PHONY: generate tidy build test clean build-all

generate:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 -config internal/openapi/client.cfg.yaml api/omlox-hub.v0.yaml

tidy:
	go mod tidy

build: generate
	go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) ./cmd/olh

test: generate
	go test ./...

clean:
	rm -rf $(DIST)

build-all: generate
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-amd64 ./cmd/olh
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-arm64 ./cmd/olh
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-amd64 ./cmd/olh
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-arm64 ./cmd/olh
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-windows-amd64.exe ./cmd/olh
