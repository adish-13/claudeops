.PHONY: build web go run dev clean

# Default: full production build (frontend + backend → single binary).
build: web go

# Build the React app to internal/server/dist (embedded by Go).
web:
	cd web && npm run build

# Build the Go binary (will fail if web/ hasn't been built yet).
go:
	go build -o claudeops ./cmd/claudeops

# Run the production binary (assumes already built).
run:
	./claudeops

# Dev mode: run Go backend on :7777 + Vite dev server on :5173 (with HMR).
# Open http://localhost:5173 — it proxies API+WS calls to :7777.
dev:
	@echo "Start backend in one terminal: ./claudeops"
	@echo "Then start frontend HMR:       cd web && npm run dev"

clean:
	rm -f claudeops
	rm -rf internal/server/dist
	rm -rf web/node_modules
