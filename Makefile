.PHONY: all build test test-coverage e2e benchmark lint clean docker-up docker-down contracts

# Default target
all: build test

# Build binaries
build:
	@echo "Building demo..."
	@mkdir -p bin
	go build -o bin/demo ./cmd/demo
	@echo "Building benchmark..."
	go build -o bin/benchmark ./cmd/benchmark

# Run unit tests
test:
	go test -v -race ./pkg/...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run E2E tests (requires Katana)
e2e:
	./scripts/run-e2e.sh

# Run benchmarks
benchmark:
	go run ./cmd/benchmark

# Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html data/ tmp/
	rm -rf contracts/target/

# Docker commands
docker-up:
	docker-compose -f docker/docker-compose.yml up -d

docker-down:
	docker-compose -f docker/docker-compose.yml down

docker-build:
	docker-compose -f docker/docker-compose.yml build

# Cairo contracts
contracts:
	cd contracts && scarb build

contracts-test:
	cd contracts && scarb test

# Deploy contracts to Katana
deploy:
	./scripts/deploy-contracts.sh

# Quick demo run
demo: build
	./bin/demo

# Development helpers
dev-setup:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Verify dependencies
verify:
	go mod verify

# Run everything (for CI)
ci: lint test-coverage build
	@echo "CI checks passed!"
