VERSION  ?= 0.008 Alpha
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

.PHONY: all clean fmt lint test \
	build build-tui \
	build-gui build-tui \
	build-plugins \
	install install-tui uninstall uninstall-tui release


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

## lint: go vet (golangci-lint temporarily disabled)
lint:
	@echo "Linting..."
	go vet ./...

## test: go test -race -cover
test:
	@echo "Testing..."
	go test -v -race -cover ./...

## build: build all GUI binaries
build: $(addprefix build-,$(GUI_BINS))
	@echo "All GUI binaries built in $(BUILD_DIR)/."
	@cp -r src/lang $(BUILD_DIR)/lang 2>/dev/null || true

## build-tui: build all TUI-only binaries (CGO-free)
build-tui: build-sysman-tui build-serman-tui build-ugsman-tui build-infman-tui build-srcman-tui build-pkgman-tui build-vmsman-tui
	@echo "All TUI binaries built in $(BUILD_DIR)/."

## Generic GUI build rule
build-%:
	@echo "Building $* GUI..."
	@mkdir -p $(BUILD_DIR)
	go build -buildmode=pie -ldflags="$(PIE_LDFLAGS)" -o $(BUILD_DIR)/$* ./src/cmd/$*-gui/
	@cp -r src/lang $(BUILD_DIR)/lang 2>/dev/null || true

## Generic TUI build rule
build-%-tui:
	@echo "Building $*-tui TUI..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags tui_only -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$*-tui ./src/cmd/$*-tui/
	@cp -r src/lang $(BUILD_DIR)/lang 2>/dev/null || true

## build-plugins: build all dynamic plugin .so files
build-plugins:
	@echo "Building dynamic plugins..."
	@mkdir -p $(BUILD_DIR)/plugins
	go build -buildmode=plugin -o $(BUILD_DIR)/plugins/serman.so    ./src/pluginentry/serman/
	go build -buildmode=plugin -o $(BUILD_DIR)/plugins/xbps-src.so ./src/pluginentry/xbps_src/
	go build -buildmode=plugin -o $(BUILD_DIR)/plugins/xbps-pkg.so ./src/pluginentry/xbps_pkg/
	go build -buildmode=plugin -o $(BUILD_DIR)/plugins/vmsman.so   ./src/pluginentry/vmsman/
	go build -buildmode=plugin -o $(BUILD_DIR)/plugins/sysinfo.so  ./src/pluginentry/sysinfo/
	@echo "Plugins ready in $(BUILD_DIR)/plugins/"

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
