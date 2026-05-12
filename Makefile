.PHONY: all build web go run dev test test-go test-web lint lint-go lint-web fmt vet ci tidy clean

# Default target: full production build (frontend + backend → single binary).
all: build

build: web go

# ---- build ----

# Build the React app to internal/server/dist (embedded by Go).
web: web/node_modules
	cd web && npm run build

# Build the Go binary. Depends on `web` so the embed.FS isn't stale.
go:
	go build -o claudeops ./cmd/claudeops

web/node_modules: web/package.json web/package-lock.json
	cd web && npm install
	@touch web/node_modules

run:
	./claudeops

# Dev mode hint — two terminals.
dev:
	@echo "Backend (one terminal):  ./claudeops"
	@echo "Frontend HMR (another):  cd web && npm run dev"
	@echo "Then open http://localhost:5173"

# ---- test ----

# Full test suite — Go unit/integration + frontend type-check.
test: test-go test-web

# Go tests with race detector. -count=1 disables result caching for CI clarity.
test-go:
	go test -race -count=1 ./...

# Frontend type-check (no runtime, just tsc).
test-web: web/node_modules
	cd web && npx tsc -b

# ---- lint ----

lint: lint-go lint-web

lint-go:
	@out=$$(gofmt -l . 2>&1); \
	if [ -n "$$out" ]; then echo "gofmt issues:"; echo "$$out"; exit 1; fi
	go vet ./...

lint-web: web/node_modules
	cd web && npx tsc -b --noEmit

vet:
	go vet ./...

fmt:
	gofmt -w .

# ---- ci ----

# What CI runs: lint everything, run all tests, do a full build.
# Fail fast on any step.
ci: lint test build

tidy:
	go mod tidy
	cd web && npm install

clean:
	rm -f claudeops
	rm -rf internal/server/dist
	rm -rf web/node_modules web/dist
