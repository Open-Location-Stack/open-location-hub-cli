set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

module := "github.com/formation-res/open-location-hub-cli"
binary := env_var_or_default("BINARY", "olh")
dist := env_var_or_default("DIST", "dist")
version := env_var_or_default("VERSION", `git describe --tags --always --dirty 2>/dev/null || echo dev`)
commit := env_var_or_default("COMMIT", `git rev-parse --short HEAD 2>/dev/null || echo unknown`)
date := env_var_or_default("DATE", `date -u +"%Y-%m-%dT%H:%M:%SZ"`)
default:
  @just --list

generate:
  go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 -config internal/openapi/client.cfg.yaml api/omlox-hub.v0.yaml

tidy:
  go mod tidy

build: generate
  mkdir -p {{dist}}
  go build -ldflags "-s -w -X {{module}}/internal/build.Version={{version}} -X {{module}}/internal/build.Commit={{commit}} -X {{module}}/internal/build.Date={{date}}" -o {{dist}}/{{binary}} ./cmd/olh

test: generate
  go test ./...

clean:
  rm -rf {{dist}}

package-release: build-all
  rm -rf release
  mkdir -p release
  for file in {{dist}}/*; do \
    base="$(basename "$file")"; \
    stem="${base%.exe}"; \
    versioned="${stem}-{{version}}"; \
    if [[ "$base" == *.exe ]]; then \
      zip -j "release/${versioned}.zip" "$file"; \
      continue; \
    fi; \
    tar -C {{dist}} -czf "release/${versioned}.tar.gz" "$base"; \
  done
  ( \
    cd release; \
    if command -v sha256sum >/dev/null 2>&1; then \
      sha256sum * > checksums.txt; \
    else \
      shasum -a 256 * > checksums.txt; \
    fi \
  )

build-all: generate
  mkdir -p {{dist}}
  GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X {{module}}/internal/build.Version={{version}} -X {{module}}/internal/build.Commit={{commit}} -X {{module}}/internal/build.Date={{date}}" -o {{dist}}/{{binary}}-darwin-amd64 ./cmd/olh
  GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X {{module}}/internal/build.Version={{version}} -X {{module}}/internal/build.Commit={{commit}} -X {{module}}/internal/build.Date={{date}}" -o {{dist}}/{{binary}}-darwin-arm64 ./cmd/olh
  GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X {{module}}/internal/build.Version={{version}} -X {{module}}/internal/build.Commit={{commit}} -X {{module}}/internal/build.Date={{date}}" -o {{dist}}/{{binary}}-linux-amd64 ./cmd/olh
  GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X {{module}}/internal/build.Version={{version}} -X {{module}}/internal/build.Commit={{commit}} -X {{module}}/internal/build.Date={{date}}" -o {{dist}}/{{binary}}-linux-arm64 ./cmd/olh
  GOOS=windows GOARCH=386 go build -ldflags "-s -w -X {{module}}/internal/build.Version={{version}} -X {{module}}/internal/build.Commit={{commit}} -X {{module}}/internal/build.Date={{date}}" -o {{dist}}/{{binary}}-windows-386.exe ./cmd/olh
  GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X {{module}}/internal/build.Version={{version}} -X {{module}}/internal/build.Commit={{commit}} -X {{module}}/internal/build.Date={{date}}" -o {{dist}}/{{binary}}-windows-amd64.exe ./cmd/olh
