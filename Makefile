.PHONY: build test test-sdk test-e2e test-all lint validate docker-up docker-down sdk-check

# Auto-detect container runtime (podman preferred over docker)
CONTAINER_RT := $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)
COMPOSE_CMD  := $(shell if command -v podman-compose >/dev/null 2>&1; then echo podman-compose; elif command -v podman >/dev/null 2>&1 && podman compose version >/dev/null 2>&1; then echo podman compose; elif command -v docker >/dev/null 2>&1; then echo docker compose; fi)

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

# Run Go SDK tests
test-sdk:
	@echo "Running Go SDK tests..."
	go test ./sdk/backd-go/...

# Run E2E tests (requires docker-compose stack)
test-e2e:
	@echo "Running E2E tests..."
	go test ./tests/e2e/... -tags e2e -v -timeout 120s

# Run all tests (unit + SDK)
test-all:
	@echo "Running all tests..."
	go test ./internal/... ./sdk/backd-go/...

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
	@echo "Starting development stack ($(COMPOSE_CMD))..."
	$(COMPOSE_CMD) up -d

# Stop development stack
docker-down:
	@echo "Stopping development stack ($(COMPOSE_CMD))..."
	$(COMPOSE_CMD) down

# Logs development stack
docker-logs:
	@echo "Logs development stack ($(COMPOSE_CMD))..."
	$(COMPOSE_CMD) logs -f

# Check TypeScript SDK
sdk-check:
	@echo "Checking TypeScript SDK..."
	deno check sdk/backd-js/src/index.ts sdk/backd-js/src/deno.ts
