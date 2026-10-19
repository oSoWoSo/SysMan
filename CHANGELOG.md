# Changelog

All notable changes to SysMan are documented here.

---

## [Unreleased]

No unreleased changes.

---

## [0.024 Alpha]

### Added
- **pkgman: Flatpak support** — browse Flathub apps and install, remove, and update them alongside xbps and AppImage packages
- **infman: additional system details** — load average, swap, GPU, init system, xbps version, and root filesystem usage
- **pkgman: repository management in settings** — configure the xbps repositories file and manage the user repository pool, with signing-key confirmation before enabling a repository

### Changed
- **pkgman: backend selection** — choose xbps, AppImage, and Flatpak package sources independently
- **pkgman: AppImage links** — use the current Portable Linux Apps URL format
- **common: settings dialogs** — long forms are now scrollable and use a larger default size

---

## [0.023.1 Alpha]

### Fixed
- **Lint findings** — removed unused service GUI fields and corrected ANSI block-segment formatting

---

## [0.023 Alpha]

### Added
- **vmsman: automatic state refresh** — the GUI polls VM status every three seconds, including changes made outside vmsman

### Fixed
- **vmsman: quickemu PID handling** — rely on quickemu/QEMU to write the VM PID file; writing the boot wrapper PID there prevented quickemu from starting

---

## [0.022 Alpha]

### Changed
- **Maintenance release** — formatting and lint fixes

---

## [0.021 Alpha]

### Added
- **vmsman: VM creation and configuration editing** — create quickemu VMs in the GUI or TUI and edit their configuration
- **vmsman: live boot logs and SSH access** — stream boot output to the GUI/TUI and log file; configure a per-VM SSH user in the GUI or config and connect over SSH
- **pkgman: AppImage improvements** — configure the applications directory, show installed app sizes, and apply install/remove actions directly to the selected package
- **pkgman: custom xbps repositories** — maintain a user repository pool and enable or disable repositories with signature-trust confirmation
- **`common.NewApp()`** — create Fyne apps with a shared application ID

### Changed
- **vmsman: non-blocking boot and graceful stop** — boot output no longer blocks the interface; stopping requests a clean quickemu shutdown before falling back to signals
- **Charm TUI libraries upgraded to v2** and Bubble Tea v2 APIs adopted

### Fixed
- **vmsman: VM state detection and SSH connection** — read quickemu state files from each VM's directory, parse SSH/SPICE port records, and keep the terminal open to show SSH errors
- **vmsman: VM status display** — populate the GUI count row, align detail rows, and use consistent running/stopped indicators
- **pkgman: AppImage size display** — show disk usage only for installed apps

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
