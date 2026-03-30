.PHONY: build test lint clean release

VERSION ?= dev
GOOS ?= linux
GOARCH ?= amd64

build:
	@echo "Building svman ($(GOOS)/$(GOARCH))..."
	go build -ldflags="-s -w -X codeberg.org/oSoWoSo/svman/plugin.Version=$(VERSION)" -o build/svman .
	cp -r lang build/lang

test:
	@echo "Running tests..."
	go test -v -race -cover ./...

lint:
	@echo "Running linter..."
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	gofmt -s -w .

clean:
	@echo "Cleaning..."
	rm -rf build/
	go clean

release:
	@echo "Building release binary ($(GOOS)/$(GOARCH))..."
	@mkdir -p build
	go build -ldflags="-s -w -X codeberg.org/oSoWoSo/svman/plugin.Version=$(VERSION)" \
		-o build/svman-$(GOOS)-$(GOARCH) .
	cp -r lang build/lang
	cd build && sha256sum svman-$(GOOS)-$(GOARCH) > svman-$(GOOS)-$(GOARCH).sha256
	cd build && tar czf svman-$(GOOS)-$(GOARCH).tar.gz svman-$(GOOS)-$(GOARCH)
	@echo "Release binary ready in build/"

all: clean lint test build
