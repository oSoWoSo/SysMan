# Changelog

All notable changes to SysMan are documented here.

---

## [Unreleased]

### Added
- **`common.NewApp()` helper** — unified Fyne app creation with shared AppID

### Changed
- **Go modules updated** — fyne 2.8.0, fsnotify 1.10.1, x/term 0.45.0, Go 1.25
- **Charm stack upgraded to v2** — bubbletea, bubbles, lipgloss
- **Version bumped to 0.019 Alpha**

### Fixed
- **Fyne preferences error** — apps now use a unique application ID
- **Fyne threading** — `fyne.Do` migration declared
- **TUI migration for Bubble Tea v2** — updated key handling and view API

---

## [0.020 Alpha]

### Added
- **User services support (serman)** — system and user runit services with a single scope toggle
- **Per-user service layouts** — `~/.config/service` definitions and `~/service` live dir
- **Configurable default scope** — `default_scope: system|user` in config or module settings
- **i18n** — new group/detail/scope keys in EN and CS

### Changed
- **Backend interface** — actions take `Service` (with scope) instead of a bare name
- **Status maps** — keyed by `Service.Key()` (scope-prefixed)
- **Elevation** — only for system-service changes and explicit status refresh
- **Scope filters** — All/Enabled/Disabled always respect the active scope
- **Version bumped to 0.020 Alpha**

### Fixed
- **serman-tui entry point** — removed duplicate `InitI18n()` call
- **TUI nil-pointer** — selection guarded when the filtered list is empty
- **GUI detail panel** — auto-selects the first service instead of staying empty

---

## [0.014 Alpha]

### Added
- **Makefile help** — `make` and `make help` list all targets
- **Per-module release tarballs** — each binary ships with its own lang/ directory
- **XBPS template** — proper subpackages for all binaries
- **Unit tests for src/common**

### Changed
- **Default target** — bare `make` shows help instead of building
- **golangci-lint** — pinned to v2.1.1 in CI

### Fixed
- **Makefile** — removed broken build-plugins target, fixed duplicate .PHONY
- **release.yml** — updated module names

---

## [0.013 Alpha]

### Added
- **ForkURL** — personal void-packages clone support in srcman settings
- **LangDir** — configurable language directory in settings
- **Version mismatch fix** — docs updated for v0.009+
- **Module naming** — docs updated to the new module names

### Fixed
- **Settings panel** — rebuilds on click, preserves LangDir on save
- **Duplicate config** — removed duplicate serman config from settings
- **SearchEngine** — fixed default value in srcman settings
- **Language files** — serman uses "serman" instead of "svman"

---

## [0.009 Alpha]

### Added
- **Users & Groups plugin** — new `ugsman` tab and standalone binaries
- **Batch service status** — single elevated call for all enabled services
- **Reload hint** — reminder to reload for current status
- **Esc to quit** — in all standalone GUI windows
- **Button icons** — in Services and xbps-src GUI
- **Root warning** — on the Reload button when not running as root
- **`StatusAll`** — batched status queries in the Backend interface
- **Dedicated cmd entry points** — per tool/mode combination
- **vmsman plugin** — QEMU VM manager (GUI + TUI, SPICE support)
- **ANSI color support** — PTY-based output in xbps and xbps-src GUI
- **HoverableButton** — button with hover status text
- **Tooltip translations** — for all managers
- **srcman build modes** — `-Q` (tests) and `-C` (confpkg)
- **i18n tests** — for all modules
- **Module-specific tooltip keys** — `tooltip.<module>.*` prefix
- **InitI18n() in main()** — called before UI creation in all entry points

### Changed
- **Static website** — new site with 50+ retro themes
- **GUI binaries accept `--tui`/`--gui`/`--auto`** — auto-detect display
- **Binary naming** — GUI binaries drop `-gui`, TUI-only add `-tui`
- **Makefile** — restructured with per-binary targets
- **`sv status`** — run with elevation for accurate state display
- **Module path** — `codeberg.org/oSoWoSo/SysMan`
- **Directory structure** — reorganized into `src/` and `web/`
- **Module naming** — unified convention (serman, pkgman, srcman, infman, ugsman, vmsman)
- **Generic Filter** — reusable `Filter[T]` in each plugin
- **Go dependencies** — updated
- **golangci-lint** — updated configuration

### Fixed
- **ugman-tui scrolling** — list clipped to terminal height
- **ugman-tui quit key** — added `q`, `Esc`, `Ctrl+C`
- **usergroups plugin** — compile check broke `tui_only` builds
- **Fyne threading** — `fyne.Do()` wrappers in pkgman, srcman
- **Code review fixes** — simplified highlight, cleanups in sysinfo and serman TUI
- **langDirs paths** — corrected language directory resolution
- **vmsman plugin** — fixed config loading and i18n initialization
- **Tooltip translations** — show translated text instead of key names
- **Makefile lang directory** — preserved module subdirectories

### Security
- **PIE build** — for `sysman` and `ugman` binaries

---

## [0.008 Alpha]

### Added
- **Common package** — shared helpers for all plugins
- **vmsman tooltips** — Czech and English translations
- **Window icons** — set on startup

### Changed
- **Package consolidation** — `src/config/` and `src/tui/` merged into `src/common/`
- **Version management** — ldflags target `common.Version`
- **HoverableButton / ShowAbout** — used in all GUI modules
- **sysman settings** — config field names fixed
- **infman** — logo loading via `common.LogoImage()`

### Fixed
- **Import cycle** — ANSI constants moved to `common`
- **sysman settings** — `dialog.ShowError` receives an error

---

## Earlier history

See `git log` for full history prior to the changelog being introduced.