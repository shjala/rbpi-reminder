# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary name and directory
BINARY_NAME=simple-reminder
HASH_PASSWORD_NAME=hash-password
SRC_DIR=./src
TOOLS_DIR=./tools
BIN_DIR=bin
BINARY_PATH=$(BIN_DIR)/$(BINARY_NAME)
HASH_PASSWORD_PATH=$(BIN_DIR)/$(HASH_PASSWORD_NAME)

.PHONY: all build clean test deps run help build-tools build-hash-password install

# Default target
all: build build-tools

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BINARY_PATH) $(SRC_DIR)
	@echo "Build completed successfully!"

# Build tools
build-tools: build-hash-password

# Build hash-password tool
build-hash-password:
	@echo "Building $(HASH_PASSWORD_NAME) tool..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(HASH_PASSWORD_PATH) $(TOOLS_DIR)/hash-password.go
	@echo "Hash-password tool build completed successfully!"

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_PATH)
	rm -f $(HASH_PASSWORD_PATH)
	rm -rf $(BIN_DIR)
	@echo "Clean completed!"

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_PATH)

install:
	@echo "Running install script..."
	./install.sh

help:
	@echo "Available targets:"
	@echo "  build             - Build the application for current platform"
	@echo "  build-tools       - Build all tools (hash-password)"
	@echo "  clean             - Clean build artifacts"
	@echo "  test              - Run tests"
	@echo "  deps              - Download and tidy dependencies"
	@echo "  run               - Build and run the application"
	@echo "  install           - Install binary, resources, and user systemd service"
	@echo "  help              - Show this help message"
