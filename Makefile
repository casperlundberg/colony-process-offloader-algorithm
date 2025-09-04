# CAPE - Colony Adaptive Process Engine
# Makefile for building and running the project

.PHONY: all build clean run test spike-sim help

# Variables
BINARY_DIR=bin
CMD_DIR=cmd
PKG_DIR=pkg
INTERNAL_DIR=internal
BINARY_NAME=cape-simulator
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)

# Default target
all: build

# Build all binaries
build:
	@echo "Building cape-simulator..."
	@mkdir -p $(BINARY_DIR)
	@go build -o $(BINARY_PATH) ./$(CMD_DIR)

# Run spike simulation with default parameters
run: build
	@echo "Running spike simulation (1 hour)..."
	@$(BINARY_PATH) -hours 1

# Run full spike simulation (168 hours)
run-full: build
	@echo "Running full spike simulation (168 hours)..."
	@$(BINARY_PATH) -hours 168 | tee results/simulation_$(shell date +%s).log

# Run spike simulation with custom parameters
spike-sim: build
	@echo "Running spike simulation with custom parameters..."
	@echo "Usage: make spike-sim ARGS='-hours 24'"
	@$(BINARY_PATH) $(ARGS)

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BINARY_DIR)/*
	@go clean

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run || echo "golangci-lint not installed, skipping..."

# Show help
help:
	@echo "CAPE - Colony Adaptive Process Engine"
	@echo ""
	@echo "Available targets:"
	@echo "  make build          - Build all binaries"
	@echo "  make run            - Run spike simulation (1 hour)"
	@echo "  make run-full       - Run full spike simulation (168 hours)"
	@echo "  make spike-sim      - Run spike simulation with custom args"
	@echo "  make test           - Run all tests"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Lint code"
	@echo "  make help           - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make spike-sim ARGS='-hours 24'                    # Run 24-hour simulation"
	@echo "  make spike-sim ARGS='-hours 4 -verbose'            # Run 4-hour verbose simulation"