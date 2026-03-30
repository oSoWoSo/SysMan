VERSION  ?= 0.003 Alpha
GOOS     ?= linux
GOARCH   ?= amd64
PREFIX   ?= /usr/local
LDFLAGS   = -s -w -X 'codeberg.org/oSoWoSo/SysMan/plugin.Version=$(VERSION)'

# add PIE flags for cgo builds
CGO_ENABLED ?= 1
CC ?= gcc
PIE_EXTLDFLAGS = -linkmode=external -extldflags "-Wl,-pie"
PIE_LDFLAGS = $(LDFLAGS) $(PIE_EXTLDFLAGS)

BUILD_DIR = build

GUI_BINS = sysman svman ugman infoman srcman pkgman
TUI_BINS = sysman-tui svman-tui ugman-tui infoman-tui srcman-tui pkgman-tui

.PHONY: all clean fmt lint test \
	build build-tui \
	build-sysman   build-sysman-tui \
	build-svman    build-svman-tui  \
	build-ugman    build-ugman-tui  \
	build-infoman  build-infoman-tui \
	build-srcman   build-srcman-tui  \
	build-pkgman   build-pkgman-tui  \
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

## lint: go vet + golangci-lint
lint:
	@echo "Linting..."
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

## test: go test -race -cover
test:
	@echo "Testing..."
	go test -v -race -cover ./...

## build: build all GUI binaries (TUI entry points included)
build: build-sysman build-svman build-ugman build-infoman build-srcman build-pkgman
	@echo "All GUI binaries built in $(BUILD_DIR)/."

## build-tui: build all TUI-only binaries (CGO-free)
build-tui: build-sysman-tui build-svman-tui build-ugman-tui \
           build-infoman-tui build-srcman-tui build-pkgman-tui
	@echo "All TUI binaries built in $(BUILD_DIR)/."

## build-sysman: GUI system manager embedding all plugins (also supports --tui)
build-sysman:
	@echo "Building sysman..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/sysman ./cmd/sysmanager/
	@cp -r lang $(BUILD_DIR)/lang
	@[ -f void-transparent.png ] && cp void-transparent.png $(BUILD_DIR)/ || true

## build-sysman-tui: TUI-only system manager (CGO-free)
build-sysman-tui:
	@echo "Building sysman-tui..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags tui_only -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/sysman-tui ./cmd/sysmanager/
	@cp -r lang $(BUILD_DIR)/lang

## build-svman: standalone svman GUI (also supports --tui)
build-svman:
	@echo "Building svman..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/svman .
	@cp -r lang $(BUILD_DIR)/lang

## build-svman-tui: standalone svman TUI only (CGO-free)
build-svman-tui:
	@echo "Building svman-tui..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags tui_only -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/svman-tui ./cmd/svman-tui/
	@cp -r lang $(BUILD_DIR)/lang

## build-ugman: standalone ugman GUI
build-ugman:
	@echo "Building ugman..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) CC=$(CC) \
		go build -trimpath -ldflags="$(PIE_LDFLAGS)" \
		-o $(BUILD_DIR)/ugman ./cmd/ugman-gui/

## build-ugman-tui: standalone ugman TUI only (CGO-free)
build-ugman-tui:
	@echo "Building ugman-tui..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags tui_only -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/ugman-tui ./cmd/ugman-tui/

## build-infoman: standalone infoman GUI (also supports --tui)
build-infoman:
	@echo "Building infoman..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/infoman ./cmd/infoman-gui/
	@[ -f void-transparent.png ] && cp void-transparent.png $(BUILD_DIR)/ || true

## build-infoman-tui: standalone infoman TUI only (CGO-free)
build-infoman-tui:
	@echo "Building infoman-tui..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags tui_only -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/infoman-tui ./cmd/infoman-tui/

## build-srcman: standalone srcman GUI (also supports --tui)
build-srcman:
	@echo "Building srcman..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/srcman ./cmd/srcman-gui/

## build-srcman-tui: standalone srcman TUI only (CGO-free)
build-srcman-tui:
	@echo "Building srcman-tui..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags tui_only -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/srcman-tui ./cmd/srcman-tui/

## build-pkgman: standalone pkgman GUI
build-pkgman:
	@echo "Building pkgman..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/pkgman ./cmd/pkgman-gui/

## build-pkgman-tui: standalone pkgman TUI only (CGO-free)
build-pkgman-tui:
	@echo "Building pkgman-tui..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags tui_only -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/pkgman-tui ./cmd/xbps-pkg/

## build-plugins: build all dynamic plugin .so files
build-plugins:
	@echo "Building dynamic plugins..."
	@mkdir -p $(BUILD_DIR)/plugins
	go build -buildmode=plugin -o $(BUILD_DIR)/plugins/svman.so    ./pluginentry/svman/
	go build -buildmode=plugin -o $(BUILD_DIR)/plugins/xbps-src.so ./pluginentry/xbps-src/
	go build -buildmode=plugin -o $(BUILD_DIR)/plugins/xbps-pkg.so ./pluginentry/xbps-pkg/
	go build -buildmode=plugin -o $(BUILD_DIR)/plugins/sysinfo.so  ./pluginentry/sysinfo/
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
	cp -r lang/. $(DESTDIR)$(PREFIX)/share/SysMan/lang/
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
	cp -r lang/. $(DESTDIR)$(PREFIX)/share/SysMan/lang/
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
