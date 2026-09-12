VERSION  ?= 0.023 Alpha
GOOS     ?= linux
GOARCH   ?= amd64
PREFIX   ?= /usr/local
LDFLAGS   = -s -w -X 'codeberg.org/oSoWoSo/SysMan/src/common.Version=$(VERSION)'

# add PIE flags for cgo builds
CGO_ENABLED ?= 1
CC ?= gcc
PIE_EXTLDFLAGS = -linkmode=external -extldflags "-Wl,-pie"
PIE_LDFLAGS = $(LDFLAGS) $(PIE_EXTLDFLAGS)

BUILD_DIR = build

GUI_BINS = sysman serman ugsman infman srcman pkgman vmsman
TUI_BINS = sysman-tui serman-tui ugsman-tui infman-tui srcman-tui pkgman-tui vmsman-tui

# pkg-config modules required for GUI (CGO) builds on Linux
PKG_CONFIG_MODULES = x11 xrandr xinerama xcursor xi xxf86vm gl wayland-client wayland-cursor wayland-egl xkbcommon

.PHONY: all clean fmt lint test check-deps \
	build build-tui \
	install install-tui install-all uninstall uninstall-tui uninstall-all release \
	help default

# Default target - show help when no target specified
default: help

## help: show available targets
help:
	@echo "SysMan Makefile targets:"
	@echo ""
	@echo "=== Build Targets ==="
	@echo "  make build              - Build all GUI binaries (sysman, serman, pkgman, srcman, infman, ugsman, vmsman)"
	@echo "  make build-tui          - Build all TUI binaries (sysman-tui, serman-tui, ...)"
	@echo ""
	@echo "  make build-<module>     - Build single GUI binary"
	@echo "    Examples: make build-serman, make build-pkgman, make build-srcman"
	@echo "    Available: sysman, serman, pkgman, srcman, infman, ugsman, vmsman"
	@echo ""
	@echo "  make build-<module>-tui - Build single TUI binary"
	@echo "    Examples: make build-serman-tui, make build-pkgman-tui"
	@echo ""
	@echo "=== Other Targets ==="
	@echo "  make all                - clean → lint → test → build"
	@echo "  make clean              - remove build artefacts"
	@echo "  make fmt                - gofmt -s"
	@echo "  make lint               - golangci-lint"
	@echo "  make test               - go test -race -cover"
	@echo "  make check-deps         - verify pkg-config modules for GUI builds"
	@echo "  make install            - install binaries to \$$PREFIX/bin"
	@echo "  make install-tui        - install TUI binaries"
	@echo "  make install-all        - install all binaries (GUI + TUI)"
	@echo "  make uninstall          - remove installed binaries"
	@echo "  make uninstall-tui      - remove installed TUI binaries"
	@echo "  make uninstall-all      - remove all installed binaries (GUI + TUI)"
	@echo "  make release            - build all + create tarballs with checksums"
	@echo ""
	@echo "=== Variables ==="
	@echo "  VERSION=$(VERSION)   - override version (default)"
	@echo "  PREFIX=/usr/local       - installation prefix (default: /usr/local)"
	@echo "  DESTDIR=/               - staging directory for make install"
	@echo ""
	@echo "Run 'make <target>' to execute a specific target."


## all: clean → lint → test → build
all: clean lint fmt test build

## clean: remove build artefacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)/
	go clean

## fmt: gofmt -s
fmt:
	@echo "Formatting..."
	gofmt -s -w .

## lint: golangci-lint
lint:
	@echo "Linting..."
	golangci-lint run

## test: go test -race -cover
test:
	@echo "Testing..."
	go test -v -race -cover ./...

## check-deps: verify pkg-config modules required for GUI builds
check-deps:
ifeq ($(GOOS),linux)
	@command -v pkg-config >/dev/null 2>&1 || { echo "ERROR: pkg-config not found. Install: sudo xbps-install pkg-config"; exit 1; }
	@if ! pkg-config --exists --print-errors $(PKG_CONFIG_MODULES); then \
		echo ""; \
		echo "Missing GUI build dependencies. On Void Linux:"; \
		echo "  sudo xbps-install libX11-devel libXrandr-devel libXinerama-devel libXcursor-devel libXi-devel libXxf86vm-devel MesaLib-devel wayland-devel libxkbcommon-devel libglvnd-devel"; \
		exit 1; \
	fi
	@echo "check-deps: OK"
else
	@echo "check-deps: skipped (GOOS=$(GOOS))"
endif

## build: build all GUI binaries
build: $(addprefix build-,$(GUI_BINS))
	@echo "All GUI binaries built in $(BUILD_DIR)/."
	@cp -r src/lang/. $(BUILD_DIR)/lang 2>/dev/null || true

## build-tui: build all TUI-only binaries (CGO-free)
build-tui: $(addprefix build-,$(TUI_BINS))
	@echo "All TUI binaries built in $(BUILD_DIR)/."

## Generic GUI build rule
build-%: check-deps
	@echo "Building $* GUI..."
	@mkdir -p $(BUILD_DIR)
	go build -buildmode=pie -ldflags="$(PIE_LDFLAGS)" -o $(BUILD_DIR)/$* ./src/cmd/$*-gui/
	@cp -r src/lang/. $(BUILD_DIR)/lang 2>/dev/null || true

## Generic TUI build rule
build-%-tui:
	@echo "Building $*-tui TUI..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags tui_only -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$*-tui ./src/cmd/$*-tui/
	@cp -r src/lang/. $(BUILD_DIR)/lang 2>/dev/null || true

## install: build and install all GUI binaries and data files
install: build
	@echo "Installing to $(DESTDIR)$(PREFIX)/bin/ ..."
	@for bin in $(GUI_BINS); do \
	    if [ -f $(BUILD_DIR)/$$bin ]; then \
	        install -Dm755 $(BUILD_DIR)/$$bin $(DESTDIR)$(PREFIX)/bin/$$bin; \
	        echo "  installed $$bin"; \
	    fi; \
	done
	@echo "Installing lang files to $(DESTDIR)$(PREFIX)/share/SysMan/lang/ ..."
	install -d $(DESTDIR)$(PREFIX)/share/SysMan/lang
	cp -r src/lang/. $(DESTDIR)$(PREFIX)/share/SysMan/lang/
	@[ -f void-transparent.png ] && \
	    install -Dm644 void-transparent.png $(DESTDIR)$(PREFIX)/share/SysMan/ || true
	@echo "Done. highlight.conf is created in ~/.config/SysMan/ on first run."

## install-tui: build and install all TUI-only binaries
install-tui: build-tui
	@echo "Installing TUI binaries to $(DESTDIR)$(PREFIX)/bin/ ..."
	@for bin in $(TUI_BINS); do \
	    if [ -f $(BUILD_DIR)/$$bin ]; then \
	        install -Dm755 $(BUILD_DIR)/$$bin $(DESTDIR)$(PREFIX)/bin/$$bin; \
	        echo "  installed $$bin"; \
	    fi; \
	done
	@echo "Installing lang files to $(DESTDIR)$(PREFIX)/share/SysMan/lang/ ..."
	install -d $(DESTDIR)$(PREFIX)/share/SysMan/lang
	cp -r src/lang/. $(DESTDIR)$(PREFIX)/share/SysMan/lang/
	@echo "Done."

## install-all: build and install all binaries (GUI + TUI)
install-all: install install-tui

## uninstall: remove installed GUI binaries and data files
uninstall:
	@echo "Uninstalling GUI binaries from $(DESTDIR)$(PREFIX)/bin/ ..."
	@for bin in $(GUI_BINS); do \
	    rm -f $(DESTDIR)$(PREFIX)/bin/$$bin && echo "  removed $$bin" || true; \
	done
	@echo "Removing $(DESTDIR)$(PREFIX)/share/SysMan/ ..."
	rm -rf $(DESTDIR)$(PREFIX)/share/SysMan/
	@echo "Done."

## uninstall-tui: remove installed TUI binaries
uninstall-tui:
	@echo "Uninstalling TUI binaries from $(DESTDIR)$(PREFIX)/bin/ ..."
	@for bin in $(TUI_BINS); do \
	    rm -f $(DESTDIR)$(PREFIX)/bin/$$bin && echo "  removed $$bin" || true; \
	done
	@echo "Done."

## uninstall-all: remove all installed binaries (GUI + TUI)
uninstall-all: uninstall uninstall-tui

## release: build all, create per-binary tarballs with checksums
release: build build-tui
	@echo "Creating release artefacts for $(GOOS)/$(GOARCH) ..."
	@cd $(BUILD_DIR) && for bin in $(GUI_BINS) $(TUI_BINS); do \
	    [ -f $$bin ] || continue; \
	    name=$$bin-$(VERSION)-$(GOOS)-$(GOARCH); \
	    cp $$bin $$name; \
	    sha256sum $$name > $$name.sha256; \
	    tar czf $$name.tar.gz $$name lang/ 2>/dev/null || tar czf $$name.tar.gz $$name; \
	    rm $$name; \
	    echo "  $$name.tar.gz"; \
	done
	@echo "Release artefacts ready in $(BUILD_DIR)/."
