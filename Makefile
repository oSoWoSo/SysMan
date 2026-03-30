.PHONY: build test lint clean release

VERSION ?= dev
GOOS ?= linux
GOARCH ?= amd64

build:
	@echo "Building svman ($(GOOS)/$(GOARCH))..."
	go build -ldflags="-s -w -X main.Version=$(VERSION)" -o build/svman .

test:
	@echo "Running tests..."
	go test -v -race -cover ./...

lint:
	@echo "Running linter..."
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

clean:
	@echo "Cleaning..."
	rm -rf build/
	go clean

release:
	@echo "Building release binaries..."
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o build/svman-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o build/svman-linux-arm64 .
	GOOS=freebsd GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o build/svman-freebsd-amd64 .
	@echo "✅ Release binaries ready in build/"

all: clean lint test build
