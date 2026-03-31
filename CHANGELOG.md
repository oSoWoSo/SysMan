# Changelog

All notable changes to SysMan are documented here.

---

## [0.013 Alpha]

### Added
- **ForkURL** — personal void-packages clone support in srcman settings
- **LangDir** — configurable language directory in settings panel
- **Version mismatch fix** — updated documentation to reflect v0.009+ changes
- **Module naming** — updated README.md, README-cs.md, AGENTS.md to use new module names (serman, pkgman, srcman, infman, ugsman, vmsman)

### Fixed
- **Settings panel** — rebuilds on click, preserves LangDir on save
- **Duplicate config** — removed duplicate serman config from settings
- **SearchEngine** — fixed default value in srcman settings
- **Language files** — updated serman lang files to use "serman" instead of "svman"

---

## [0.009 Alpha]

### Added
- **Users & Groups plugin** (`usergroups`) — new tab in sysman and standalone `ugman` / `ugman-tui` binaries
  - Lists users (UID, full name, primary group, home) with toggle for system users
  - Lists groups (GID, members)
  - Keyboard navigation with scrolling to fit terminal height
- **Batch service status** — Reload fetches all enabled service statuses in a single elevated call; switching between services no longer prompts for a password repeatedly
- **Reload hint** — info area in the Services tab now shows a reminder to reload for current status
- **Esc to quit** in all standalone GUI windows (`svman`, `ugman`, `infoman`, `srcman`, `pkgman`)
- **Button icons** in the Services GUI and xbps-src GUI
- **Root warning** on the Reload button in Services — shows a warning icon when not running as root
- **`StatusAll`** method added to the `Backend` interface for batched service status queries
- **Dedicated cmd entry points** for each tool/mode combination:
  - `cmd/ugman-gui/`, `cmd/ugman-tui/`
  - `cmd/pkgman-gui/`
  - `cmd/infoman-gui/`, `cmd/infoman-tui/`
  - `cmd/srcman-gui/`, `cmd/srcman-tui/`
- **vmsman plugin** — QEMU VM manager with GUI and TUI interfaces, SPICE connection support
  - Boot, kill, and connect to VMs with status tracking (PID, SPICE port)
  - Filter by running/stopped state, search by name
  - Sectioned config system with per-module settings
- **ANSI color support** in xbps and xbps-src GUI output
  - PTY-based output capture for xbps-src to enable ANSI colors from build scripts
  - ANSI escape sequence parser converts SGR codes to Fyne RichText segments
  - Supports 16-color, 256-color palette, and 24-bit true color
- **HoverableButton** component — buttons that display status text on hover
- **Tooltip translations** for all managers (infoman, pkgman, srcman, serman, ugsman)
- **srcman build mode selection** — `-Q` (with tests) and `-C` (confpkg) checkboxes for xbps-src builds
- **i18n tests** for all modules (infman, pkgman, serman, srcman, ugsman, vmsman)
- **Module-specific tooltip keys** — all modules use `tooltip.<module>.*` prefix to prevent conflicts
- **InitI18n() in main()** — all cmd entry points call `InitI18n()` for their modules before UI creation

### Changed
- **Static website** — new website with 50+ retro/nostalgic themes (Amiga 500, C64, DOS, MacOS, etc.)
- **Standalone GUI binaries support TUI mode** — `infoman`, `pkgman`, `srcman`, `ugman` accept `--tui`/`--gui`/`--auto` flags; auto-detects DISPLAY/WAYLAND_DISPLAY
- **Binary naming**: GUI binaries no longer carry a `-gui` suffix; TUI-only binaries use `-tui` suffix
  - `sysmanager` → `sysman` / `sysman-tui`
  - `svman` stays `svman` (GUI+TUI); `svman-tui` (TUI only)
  - New: `ugman`, `ugman-tui`, `infoman`, `infoman-tui`, `srcman`, `srcman-tui`, `pkgman`, `pkgman-tui`
- **Makefile** fully restructured with per-binary targets (`build-svman`, `build-svman-tui`, etc.) and `install`/`uninstall`/`release` targets
- **`sv status`** is now run with `api.Elevate` (pkexec/doas/sudo) for accurate service state display
- Module path is `codeberg.org/oSoWoSo/SysMan`
- **Directory structure** — reorganized into `src/` (Go code) and `web/` (static site)
  - All plugins moved to `src/<name>/` (serman, pkgman, srcman, infman, ugsman, vmsman)
  - All entry points moved to `src/cmd/<name>-gui/` and `src/cmd/<name>-tui/`
  - Language files moved to `src/lang/<name>/`
- **Module naming** — unified naming convention across all modules:
  - `plugin` → `serman`, `xbps-pkg` → `pkgman`, `xbps-src` → `srcman`
  - `sysinfo` → `infman`, `usergroups` → `ugsman`, `vmman` → `vmsman`
  - `svman` → `serman` (in lang files and internal references)
- **Generic Filter** — refactored filter logic into reusable generic `Filter[T]` function in each plugin
- **Go dependencies** updated to latest versions
- **golangci-lint** configuration updated with additional rules

### Fixed
- `ugman-tui`: list was not scrolled/clipped to terminal height — added sliding window scroll
- `ugman-tui`: no quit key was bound — added `q`, `Esc`, `Ctrl+C`
- `usergroups/plugin.go`: removed compile-time interface check that broke `tui_only` builds
- **Fyne threading** — wrapped UI updates in `fyne.Do()` to prevent race conditions in pkgman, srcman
- **Code review fixes** — simplified highlight.go, cleaned up sysinfo, improved serman TUI
- **langDirs paths** — corrected language directory resolution paths across all modules
- **vmsman plugin** — fixed config loading and i18n initialization
- **Tooltip translations** — tooltips now show translated text instead of key names
  - Added module-specific prefixes (`tooltip.serman.*`, `tooltip.pkgman.*`, etc.) to prevent key conflicts
  - Added `InitI18n()` calls in main() for all modules before UI creation
- **Makefile lang directory** — fixed `cp -r src/lang/. $(BUILD_DIR)/lang` to preserve module subdirectories

### Security
- **PIE build** for `sysman` and `ugman` binaries — required for Void Linux packages

---

## [0.008 Alpha]

### Added
- **Common package** (`src/common/`) — unified shared helpers for all plugins:
  - `common.Filter[T]` — generic filter function replacing 3 duplicate implementations
  - `common.ShowAbout()` — standardized About dialog replacing 5 duplicate implementations
  - `common.HoverableButton` — button with hover status text replacing 4 duplicate implementations
  - `common.SetWindowIcon()` / `common.AppIcon()` / `common.LogoImage()` — application icon helpers
  - `common.Version`, `common.AppAuthor`, `common.AppLicense`, `common.AppURL` — centralized app metadata
  - `common.AnsiRe`, `common.ParseSeq`, `common.AnsiToRichSegments` — ANSI rendering helpers
- **vmsman tooltips** — Czech and English translations for boot, kill, connect, about, reload, filter, search
- **Window icons** — sysman-gui now sets application icon on startup

### Changed
- **Package consolidation** — `src/config/` and `src/tui/` merged into `src/common/`
- **Version management** — ldflags now targets `common.Version` instead of `serman.Version`
- **All GUI modules** (vmsman, serman, ugsman, infman, pkgman, srcman) use `common.HoverableButton` and `common.ShowAbout`
- **sysman settings** — fixed config field names (`Serman`, `Vmsman` instead of `Svman`, `Vmman`)
- **infman** — logo loading now delegates to `common.LogoImage()`

### Fixed
- **Import cycle** — ANSI constants moved from `infman` to `common` to break `serman → common → infman → serman` cycle
- **sysman settings** — `dialog.ShowError` now receives an `error` instead of a string

---

## Earlier history

See `git log` for full history prior to the changelog being introduced.
