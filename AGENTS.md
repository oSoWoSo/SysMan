# Repository Guidelines

## Layout and interfaces
- Go module: `codeberg.org/oSoWoSo/SysMan` (Go 1.25). Application code is under `src/`: reusable modules in `src/<module>`, shared helpers in `src/common`, entrypoints in `src/cmd`, and translations in `src/lang/<module>/{en,cs}.yaml`.
- `api.PluginIF` is TUI/core (`Name`, `Model`); GUI modules additionally implement `api.PluginGUI` (`Content`, `SetStatusBar`). Do not assume GUI methods exist in `tui_only` builds.
- Use `src/common` for shared helpers (`NewApp`, `Filter`, `ShowAbout`, `NewHoverableButton`). Create Fyne apps with `common.NewApp`; marshal UI updates from goroutines through `fyne.Do`.

## Build and test
- `make build` builds all seven GUI binaries and requires CGO plus Linux GUI development libraries (`pkg-config`, X11, Wayland, OpenGL).
- `make build-tui` builds all seven TUI binaries with `CGO_ENABLED=0` and `tui_only`; use `make build-<module>` or `make build-<module>-tui` for one binary.
- `make test` runs `go test -v -race -cover ./...`; focus a package with `go test ./src/<module>/...` or a test with `go test ./src/<module> -run '^TestName$'`.
- `make lint` runs `golangci-lint`; CI also runs `gofmt -s` verification and `go vet ./...`. `make fmt` modifies all Go files; inspect its diff before including formatting changes.

## Changes that cross modules
- For combined `sysman` entrypoints, initialize each module's `InitI18n()` before constructing UI/models. When adding an action button, add its `btn.*` label and module-prefixed `tooltip.<module>.*` translation to both English and Czech files.
- Language lookup can be overridden by `SYSMAN_LANGDIR` (highest priority); config is YAML at `~/.config/sysman/sysman.conf`.
- The Makefile is the source of truth for targets and build flags; `.forgejo/workflows/build.yml` defines CI checks. Keep generated binaries under `build/` out of source changes.
