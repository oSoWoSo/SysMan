# Changelog

All notable changes to SysMan are documented here.

---

## [Unreleased]

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

### Changed
- **Binary naming**: GUI binaries no longer carry a `-gui` suffix; TUI-only binaries use `-tui` suffix
  - `sysmanager` → `sysman` / `sysman-tui`
  - `svman` stays `svman` (GUI+TUI); `svman-tui` (TUI only)
  - New: `ugman`, `ugman-tui`, `infoman`, `infoman-tui`, `srcman`, `srcman-tui`, `pkgman`, `pkgman-tui`
- **Makefile** fully restructured with per-binary targets (`build-svman`, `build-svman-tui`, etc.) and `install`/`uninstall`/`release` targets
- **`sv status`** is now run with `api.Elevate` (pkexec/doas/sudo) for accurate service state display
- Module path is `codeberg.org/oSoWoSo/SysMan`

### Fixed
- `ugman-tui`: list was not scrolled/clipped to terminal height — added sliding window scroll
- `ugman-tui`: no quit key was bound — added `q`, `Esc`, `Ctrl+C`
- `usergroups/plugin.go`: removed compile-time interface check that broke `tui_only` builds

---

## Earlier history

See `git log` for full history prior to the changelog being introduced.
