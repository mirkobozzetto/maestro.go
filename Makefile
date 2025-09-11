.PHONY: help build test run proto clean install-tools lint fmt bench docker-up docker-down

GO := go
GOFLAGS := -v
BINARY_NAME := maestro
BUILD_DIR := ./bin

help:
	@echo "Maestro.go - Polyglot API Orchestrator"
	@echo ""
	@echo "Available commands:"
	@echo "  make build        - Build the maestro binary"
	@echo "  make test         - Run all tests"
	@echo "  make bench        - Run benchmarks"
	@echo "  make run          - Run with example workflow"
	@echo "  make proto        - Generate protobuf files"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make install-tools - Install required tools"
	@echo "  make lint         - Run linters"
	@echo "  make fmt          - Format code"
	@echo "  make docker-up    - Start docker services"
	@echo "  make docker-down  - Stop docker services"
	@echo ""
	@echo "Examples:"
	@echo "  make run WORKFLOW=examples/workflows/user_onboarding.yaml"
	@echo "  make test-coverage"

build:
	@echo "Building maestro..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/maestro
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test
	@echo "Generating coverage report..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./internal/...

run: build
	@echo "Running maestro with example workflow..."
	$(BUILD_DIR)/$(BINARY_NAME) execute $(WORKFLOW) --input '{"email":"user@example.com","name":"John Doe","plan":"premium"}'

proto:
	@echo "Generating protobuf files..."
	@mkdir -p pkg/proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto
	@echo "Protobuf generation complete"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "Clean complete"

install-tools:
	@echo "Installing required tools..."
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"

lint:
	@echo "Running linters..."
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Code formatted"

vendor:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies updated"

docker-up:
	@echo "Starting Docker services..."
	docker-compose up -d
	@echo "Services started"

docker-down:
	@echo "Stopping Docker services..."
	docker-compose down
	@echo "Services stopped"

docker-logs:
	docker-compose logs -f

serve: build
	@echo "Starting Maestro server..."
	$(BUILD_DIR)/$(BINARY_NAME) serve --port 8080 --debug

validate:
	@echo "Validating workflows..."
	@for workflow in examples/workflows/*.yaml; do \
		echo "Validating $$workflow..."; \
		$(BUILD_DIR)/$(BINARY_NAME) validate $$workflow || exit 1; \
	done
	@echo "All workflows valid"

WORKFLOW ?= examples/workflows/user_onboarding.yaml