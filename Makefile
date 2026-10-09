# UniCompute Makefile

# Variables
APP_NAME := unicompute
COORDINATOR_BIN := coordinator
WORKER_BIN := worker
BUILD_DIR := bin

# Go build flags
GOFLAGS := -v
LDFLAGS := -s -w

.PHONY: all clean build build-cli build-coordinator build-worker test vet fmt

# Default target
all: clean fmt vet build test

# Clean build directory
clean:
	@echo "==> Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)

# Build all binaries
build: build-cli build-coordinator build-worker

# Build Requester CLI
build-cli:
	@echo "==> Building CLI..."
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/unicompute

# Build Cloud Coordinator
build-coordinator:
	@echo "==> Building Coordinator..."
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(COORDINATOR_BIN) ./cmd/coordinator

# Build Worker Daemon
build-worker:
	@echo "==> Building Worker Daemon..."
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(WORKER_BIN) ./cmd/worker

# Run tests
test:
	@echo "==> Running tests..."
	go test -v ./...

# Run go vet
vet:
	@echo "==> Running go vet..."
	go vet ./...

# Format code
fmt:
	@echo "==> Formatting code..."
	go fmt ./...

# Run migrations (requires golang-migrate CLI)
migrate-up:
	@echo "==> Running migrations..."
	migrate -path internal/storage/migrations -database "$(DB_DSN)" up

migrate-down:
	@echo "==> Rolling back migrations..."
	migrate -path internal/storage/migrations -database "$(DB_DSN)" down
