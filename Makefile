.PHONY: build test lint validate docker-up docker-down sdk-check

# Build both binaries
build:
	mkdir -p build
	@echo "Building backd CLI..."
	go build -o build/backd ./cmd/backd
	@echo "Building API runtime..."
	go build -o build/api ./cmd/api
	@echo "Build complete."

# Run unit tests
test:
	@echo "Running unit tests..."
	go test ./internal/...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Validate configuration
validate:
	@echo "Validating configuration..."
	backd validate --config-dir ./testdata/apps

# Start development stack
docker-up:
	@echo "Starting development stack..."
	docker compose up -d

# Stop development stack
docker-down:
	@echo "Stopping development stack..."
	docker compose down

# Check TypeScript SDK
sdk-check:
	@echo "Checking TypeScript SDK..."
	deno check sdk/backd-js/src/index.ts sdk/backd-js/src/deno.ts
