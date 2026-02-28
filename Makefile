.PHONY: build build-tui test lint clean release build-sysmanager build-testplugin build-plugins

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

build-tui:
	@echo "Building svman-tui (TUI-only, CGO-free)..."
	@mkdir -p build
	CGO_ENABLED=0 go build -tags tui_only \
		-ldflags="-s -w -X codeberg.org/oSoWoSo/svman/plugin.Version=$(VERSION)" \
		-o build/svman-tui ./cmd/svman-tui/
	cp -r lang build/lang

build-testplugin:
	@echo "Building testplugin standalone binary..."
	@mkdir -p build
	go build -o build/testplugin ./cmd/testplugin/

build-sysmanager:
	@echo "Building sysmanager..."
	@mkdir -p build
	go build -ldflags="-s -w -X codeberg.org/oSoWoSo/svman/plugin.Version=$(VERSION)" \
		-o build/sysmanager ./cmd/sysmanager/
	cp -r lang build/lang

build-plugins:
	@echo "Building dynamic plugins (.so)..."
	@mkdir -p build/plugins
	go build -buildmode=plugin -o build/plugins/svman.so     ./pluginentry/svman/
	go build -buildmode=plugin -o build/plugins/testplugin.so ./pluginentry/testplugin/
	@echo "Plugins ready in build/plugins/"

all: clean lint test build
