# Keyline Developer Commands
# Usage examples:
#   just build
#   just run
#   just test
#   just integration
#   just lint             # lint check
#   just lint fix         # lint and auto-fix
#   just ci               # run all checks
#   just ci fix           # run all checks with auto-fix

set shell := ["bash", "-cu"]

# Variables
BINARY_DIR := "./bin"
CONFIG := "./config.yaml"
ENV := "DEVELOPMENT"

# Default target
default:
    @echo "Available recipes:"
    @just --summary

# -----------------------------
# Build and Run
# -----------------------------

build:
    @echo "🔧 Building Keyline API..."
    mkdir -p "{{BINARY_DIR}}"
    go build -o "{{BINARY_DIR}}/keyline-api" "./cmd/api"
    cd client && go build ./...

run: build
    @echo "🚀 Running Keyline API (environment={{ENV}})..."
    "{{BINARY_DIR}}/keyline-api" \
        --environment="{{ENV}}" \
        --config="{{CONFIG}}"

# -----------------------------
# Testing
# -----------------------------

test:
    @echo "🧪 Running unit tests..."
    go test -race -count=1 ./...
    cd client && go test -race -count=1 ./...

integration:
    @echo "🔬 Running integration tests..."
    go test -race -count=1 -tags=integration ./tests/integration/...

e2e:
    @echo "🛤️ Running e2e tests..."
    go test -race -count=1 -tags=e2e ./tests/e2e/...

# -----------------------------
# Linting & Formatting
# -----------------------------

lint fix="":
    @echo "🔍 Running linter..."
    if [ "{{fix}}" = "fix" ]; then \
        echo "🧹 Auto-fixing lint issues..."; \
        golangci-lint run --fix && (cd client && golangci-lint run --fix); \
    else \
        golangci-lint run && (cd client && golangci-lint run); \
    fi

fmt:
    @echo "🎨 Formatting code..."
    go fmt ./...
    cd client && go fmt ./...

arch:
    @echo "🏛️ Checking client module dependencies..."
    cd client && go run github.com/arch-go/arch-go@v1.7.0

docs:
    @echo "📚 Generating swagger docs..."
    go generate ./swag.go

# -----------------------------
# Utility
# -----------------------------

clean:
    @echo "🧹 Cleaning build artifacts..."
    rm -rf "{{BINARY_DIR}}"

# one-time dev setup after cloning: local go workspace
setup:
    test -f go.work || go work init . ./client

# -----------------------------
# Release
# -----------------------------

# tag and push a release: signed tags for the server and client modules
release version:
    #!/usr/bin/env bash
    set -euo pipefail
    test -z "$(git status --porcelain)" || { echo "working tree not clean"; exit 1; }
    git tag -s "client/v{{version}}" -m "client/v{{version}}"
    git tag -s "v{{version}}" -m "v{{version}}"
    git push origin "client/v{{version}}" "v{{version}}"

# -----------------------------
# CI Convenience
# -----------------------------

ci fix="":
    @echo "🏗️ Running full CI pipeline..."
    just fmt
    just lint {{fix}}
    just arch
    just test
    just integration
    just e2e
    @echo "✅ All checks passed."
