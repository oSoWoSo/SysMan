.PHONY: build build-all build-tui test lint clean release build-sysmanager build-sysinfo build-xbps-src build-xbps-pkg build-plugins

VERSION ?= dev
GOOS ?= linux
GOARCH ?= amd64

build:
	@echo "Building svman ($(GOOS)/$(GOARCH))..."
	go build -ldflags="-s -w -X codeberg.org/oSoWoSo/SysMan/plugin.Version=$(VERSION)" -o build/svman .
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
	go build -ldflags="-s -w -X codeberg.org/oSoWoSo/SysMan/plugin.Version=$(VERSION)" \
		-o build/svman-$(GOOS)-$(GOARCH) .
	cp -r lang build/lang
	cd build && sha256sum svman-$(GOOS)-$(GOARCH) > svman-$(GOOS)-$(GOARCH).sha256
	cd build && tar czf svman-$(GOOS)-$(GOARCH).tar.gz svman-$(GOOS)-$(GOARCH)
	@echo "Release binary ready in build/"

build-all: build build-tui build-sysmanager build-sysinfo build-xbps-src build-xbps-pkg
	@echo "All binaries built."

build-tui:
	@echo "Building svman-tui (TUI-only, CGO-free)..."
	@mkdir -p build
	CGO_ENABLED=0 go build -tags tui_only \
		-ldflags="-s -w -X codeberg.org/oSoWoSo/SysMan/plugin.Version=$(VERSION)" \
		-o build/svman-tui ./cmd/svman-tui/
	cp -r lang build/lang

build-sysinfo:
	@echo "Building sysinfo standalone binary..."
	@mkdir -p build
	go build -o build/sysinfo ./cmd/sysinfo/
	@[ -f void-transparent.png ] && cp void-transparent.png build/ || true

build-xbps-src:
	@echo "Building xbps-src standalone binary..."
	@mkdir -p build
	go build -o build/xbps-src ./cmd/xbps-src/

build-xbps-pkg:
	@echo "Building xbps package manager standalone binary..."
	@mkdir -p build
	go build -o build/xbps-pkg ./cmd/xbps-pkg/

build-sysmanager:
	@echo "Building sysmanager..."
	@mkdir -p build
	go build -ldflags="-s -w -X codeberg.org/oSoWoSo/SysMan/plugin.Version=$(VERSION)" \
		-o build/sysmanager ./cmd/sysmanager/
	cp -r lang build/lang
	@[ -f void-transparent.png ] && cp void-transparent.png build/ || true

build-plugins:
	@echo "Building dynamic plugins (.so)..."
	@mkdir -p build/plugins
	go build -buildmode=plugin -o build/plugins/svman.so    ./pluginentry/svman/
	go build -buildmode=plugin -o build/plugins/xbps-src.so ./pluginentry/xbps-src/
	go build -buildmode=plugin -o build/plugins/xbps-pkg.so ./pluginentry/xbps-pkg/
	go build -buildmode=plugin -o build/plugins/sysinfo.so  ./pluginentry/sysinfo/
	@echo "Plugins ready in build/plugins/"

all: clean lint test build
