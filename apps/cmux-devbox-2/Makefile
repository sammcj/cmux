# Makefile for cmux2 CLI (E2B version)

VERSION ?= 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Auth configuration (set via env or make arguments for production)
# These are baked into the binary via ldflags
STACK_PROJECT_ID ?=
STACK_PUBLISHABLE_CLIENT_KEY ?=
CMUX_API_URL ?=
CONVEX_SITE_URL ?=

# Package path for ldflags
AUTH_PKG := github.com/cmux-cli/cmux-devbox-2/internal/auth

# Production ldflags - strips debug info, bakes in config
LDFLAGS := -s -w \
           -X main.Version=$(VERSION) \
           -X main.Commit=$(COMMIT) \
           -X main.BuildTime=$(BUILD_TIME) \
           -X main.Mode=prod \
           -X '$(AUTH_PKG).ProjectID=$(STACK_PROJECT_ID)' \
           -X '$(AUTH_PKG).PublishableKey=$(STACK_PUBLISHABLE_CLIENT_KEY)' \
           -X '$(AUTH_PKG).CmuxURL=$(CMUX_API_URL)' \
           -X '$(AUTH_PKG).ConvexSiteURL=$(CONVEX_SITE_URL)'

.PHONY: build build-dev build-prod build-all build-npm clean test install install-dev deps npm-version npm-publish npm-publish-dry

# =============================================================================
# Development builds
# =============================================================================

build-dev:
	@echo "Building for development (Mode=dev)..."
	@echo "  Dev defaults: localhost:9779, famous-camel-162.convex.site"
	go build -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME) -X main.Mode=dev" \
		-o bin/cmux2 ./cmd/cmux
	@echo "Dev build complete: bin/cmux2"

install-dev: build-dev
	cp bin/cmux2 /usr/local/bin/cmux2

# =============================================================================
# Production builds
# =============================================================================

build: build-prod

build-prod:
	@echo "Building for production (Mode=prod)..."
	@if [ -z "$(STACK_PROJECT_ID)" ]; then echo "Error: STACK_PROJECT_ID is required"; exit 1; fi
	@if [ -z "$(STACK_PUBLISHABLE_CLIENT_KEY)" ]; then echo "Error: STACK_PUBLISHABLE_CLIENT_KEY is required"; exit 1; fi
	@if [ -z "$(CMUX_API_URL)" ]; then echo "Error: CMUX_API_URL is required"; exit 1; fi
	@if [ -z "$(CONVEX_SITE_URL)" ]; then echo "Error: CONVEX_SITE_URL is required"; exit 1; fi
	go build -ldflags "$(LDFLAGS)" -o bin/cmux2 ./cmd/cmux
	@echo "Production build complete: bin/cmux2"

install: build
	cp bin/cmux2 /usr/local/bin/cmux2

# =============================================================================
# Cross-platform builds
# =============================================================================

build-all:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/cmux2-darwin-amd64 ./cmd/cmux
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/cmux2-darwin-arm64 ./cmd/cmux
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/cmux2-linux-amd64 ./cmd/cmux
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/cmux2-linux-arm64 ./cmd/cmux
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/cmux2-windows-amd64.exe ./cmd/cmux

# =============================================================================
# NPM package builds
# =============================================================================

# Build binaries for npm packages
# Usage: make build-npm STACK_PROJECT_ID=... STACK_PUBLISHABLE_CLIENT_KEY=... CMUX_API_URL=... CONVEX_SITE_URL=...
build-npm:
	@echo "Building npm packages..."
	@if [ -z "$(STACK_PROJECT_ID)" ]; then echo "Error: STACK_PROJECT_ID is required"; exit 1; fi
	@if [ -z "$(STACK_PUBLISHABLE_CLIENT_KEY)" ]; then echo "Error: STACK_PUBLISHABLE_CLIENT_KEY is required"; exit 1; fi
	@if [ -z "$(CMUX_API_URL)" ]; then echo "Error: CMUX_API_URL is required"; exit 1; fi
	@if [ -z "$(CONVEX_SITE_URL)" ]; then echo "Error: CONVEX_SITE_URL is required"; exit 1; fi
	@mkdir -p npm/darwin-arm64/bin npm/darwin-x64/bin npm/linux-arm64/bin npm/linux-x64/bin npm/win32-x64/bin
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o npm/darwin-arm64/bin/cmux2 ./cmd/cmux
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o npm/darwin-x64/bin/cmux2 ./cmd/cmux
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o npm/linux-arm64/bin/cmux2 ./cmd/cmux
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o npm/linux-x64/bin/cmux2 ./cmd/cmux
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o npm/win32-x64/bin/cmux2.exe ./cmd/cmux
	@chmod +x npm/cmux2/bin/cmux2 2>/dev/null || true
	@echo "Done! Binaries built in npm/*/bin/"

# Update version in all npm package.json files
npm-version:
	@echo "Updating npm package versions to $(VERSION)..."
	@for pkg in cmux2 darwin-arm64 darwin-x64 linux-arm64 linux-x64 win32-x64; do \
		if [ -f "npm/$$pkg/package.json" ]; then \
			sed -i '' 's/"version": "[^"]*"/"version": "$(VERSION)"/' npm/$$pkg/package.json; \
			if [ "$$pkg" = "cmux2" ]; then \
				sed -i '' 's/"cmux2-darwin-arm64": "[^"]*"/"cmux2-darwin-arm64": "$(VERSION)"/' npm/$$pkg/package.json; \
				sed -i '' 's/"cmux2-darwin-x64": "[^"]*"/"cmux2-darwin-x64": "$(VERSION)"/' npm/$$pkg/package.json; \
				sed -i '' 's/"cmux2-linux-arm64": "[^"]*"/"cmux2-linux-arm64": "$(VERSION)"/' npm/$$pkg/package.json; \
				sed -i '' 's/"cmux2-linux-x64": "[^"]*"/"cmux2-linux-x64": "$(VERSION)"/' npm/$$pkg/package.json; \
				sed -i '' 's/"cmux2-win32-x64": "[^"]*"/"cmux2-win32-x64": "$(VERSION)"/' npm/$$pkg/package.json; \
			fi; \
		fi; \
	done
	@echo "Done!"

# Publish to npm (run build-npm first)
npm-publish: build-npm
	@echo "Publishing to npm..."
	@echo "1. Publishing platform packages..."
	cd npm/darwin-arm64 && npm publish --access public
	cd npm/darwin-x64 && npm publish --access public
	cd npm/linux-arm64 && npm publish --access public
	cd npm/linux-x64 && npm publish --access public
	cd npm/win32-x64 && npm publish --access public
	@echo "2. Publishing main package..."
	cd npm/cmux2 && npm publish --access public
	@echo "Done!"

# Dry-run publish (for testing)
npm-publish-dry: build-npm
	@echo "Dry-run publishing to npm..."
	cd npm/darwin-arm64 && npm publish --access public --dry-run
	cd npm/darwin-x64 && npm publish --access public --dry-run
	cd npm/linux-arm64 && npm publish --access public --dry-run
	cd npm/linux-x64 && npm publish --access public --dry-run
	cd npm/win32-x64 && npm publish --access public --dry-run
	cd npm/cmux2 && npm publish --access public --dry-run
	@echo "Done!"

# =============================================================================
# Utilities
# =============================================================================

clean:
	rm -rf bin/
	rm -rf npm/*/bin/cmux2 npm/*/bin/cmux2.exe

test:
	go test -v ./...

deps:
	go mod download
	go mod tidy
