# Makefile for aws-terror

# Variables
BINARY_NAME=aws-terror
VERSION=$(shell git describe --tags --always --dirty)
BUILD_DATE=$(shell date +%FT%T%z)
GIT_COMMIT=$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X github.com/katungi/aws-terror/cmd.Version=${VERSION} -X github.com/katungi/aws-terror/cmd.BuildDate=${BUILD_DATE} -X github.com/katungi/aws-terror/cmd.GitCommit=${GIT_COMMIT}"

.PHONY: all build clean test cover lint vet fmt install uninstall

# Default target
all: clean build test

# Build binary
build:
	@echo "Building aws-terror..."
	go build ${LDFLAGS} -o bin/${BINARY_NAME} .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
cover:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Install binary locally
install:
	@echo "Installing aws-terror..."
	go install ${LDFLAGS}

# Uninstall binary
uninstall:
	@echo "Uninstalling aws-terror..."
	rm -f $(GOPATH)/bin/${BINARY_NAME}

# Create release builds for multiple OS/architectures
release:
	@echo "Building release binaries..."
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-windows-amd64.exe .