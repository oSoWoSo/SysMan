# SysMan Contributor Guide

This document provides essential information for AI agents and human contributors working on the SysMan codebase.

## Project Overview

**SysMan** is a modular desktop and terminal application for Void Linux that combines service management, package management, template management, system information, user/group management, and VM management into a single tabbed interface. It is also a plugin framework — each tab is an independently embeddable component.

**Version:** 0.008 Alpha
**Module path:** `codeberg.org/oSoWoSo/SysMan`
**License:** MIT
**Author:** zenobit @ oSoWoSo.org

## Mandatory Rules for AI Agents

### 1. Always use `src/common/` for shared functionality

**NEVER** duplicate these components — they live in `src/common/`:

| Component | Purpose | Import |
|---|---|---|
| `common.Filter[T]` | Generic filter by state + search | `codeberg.org/oSoWoSo/SysMan/src/common` |
| `common.ShowAbout()` | Standardized About dialog | `codeberg.org/oSoWoSo/SysMan/src/common` |
| `common.HoverableButton` | Button with hover status text | `codeberg.org/oSoWoSo/SysMan/src/common` |
| `common.SetWindowIcon()` | Set window icon from standard paths | `codeberg.org/oSoWoSo/SysMan/src/common` |
| `common.LogoImage()` | Load distro logo for display | `codeberg.org/oSoWoSo/SysMan/src/common` |
| `common.Version` | App version (set via ldflags) | `codeberg.org/oSoWoSo/SysMan/src/common` |
| `common.AnsiToRichSegments()` | Convert ANSI text to Fyne segments | `codeberg.org/oSoWoSo/SysMan/src/common` |

Each module keeps its own `FilterMode` constants (e.g. `FilterEnabled`/`FilterDisabled` in serman, `FilterRunning`/`FilterStopped` in vmsman) but delegates filtering to `common.Filter`.

### 2. Always verify the build after every change

After any code modification:
```bash
go build ./...
```
Do not commit until the build succeeds.

### 3. Commit after every meaningful change

Each completed task must be committed immediately with a descriptive message. Do not batch multiple unrelated changes into a single commit.

### 4. Update CHANGELOG.md only at the end

**Do NOT** update `CHANGELOG.md` incrementally during development. Update it only after:
1. All planned changes are implemented
2. The build passes successfully
3. All commits are made

Then add a single comprehensive changelog entry covering all changes in the session.

### 5. Version management

- Version is set at build time via ldflags targeting `common.Version`
- Makefile line: `LDFLAGS = -s -w -X 'codeberg.org/oSoWoSo/SysMan/src/common.Version=$(VERSION)'`
- Default version in `src/common/version.go` is `"0.008 Alpha"`
- Each module re-exports version for backward compatibility: `var Version = common.Version`

### 6. All GUI action buttons must use `common.HoverableButton`

```go
btn := common.NewHoverableButton(
    t("btn.action"),        // label
    theme.SomeIcon(),       // icon
    t("tooltip.action"),    // hover status text
    statusBar,              // *widget.Label for status display
    func() { /* tapped */ },
)
```

### 7. All About dialogs must use `common.ShowAbout`

```go
common.ShowAbout(common.AboutConfig{
    Win:       win,
    Title:     t("app.title"),
    Subtitle:  t("app.subtitle"),
    Version:   common.Version,  // or module's re-exported Version
    Author:    common.AppAuthor,
    License:   common.AppLicense,
    URL:       common.AppURL,
    DialogBtn: t("btn.about"),
    CloseBtn:  t("btn.close"),
})
```

### 8. Language files must include tooltips

Every module's language files (`src/lang/<name>/en.yaml` and `src/lang/<name>/cs.yaml`) must have:
- `btn.*` keys for all action button labels
- `tooltip.*` keys for all hover status texts

### 9. Thread safety with Fyne

All UI updates from goroutines must use `fyne.Do()`:
```go
go func() {
    result := doWork()
    fyne.Do(func() {
        label.SetText(result)
    })
}()
```

## Project Structure

```
SysMan/
├── Makefile                     # Build system (PIE for GUI, version via ldflags)
├── CHANGELOG.md                 # All notable changes
├── README.md / README-cs.md     # Documentation
├── go.mod / go.sum              # Go dependencies
├── src/
│   ├── api/                     # PluginIF interface, elevator, highlight
│   ├── common/                  # Shared components (Filter, ShowAbout, HoverableButton, etc.)
│   │   ├── about.go             # ShowAbout + AboutConfig
│   │   ├── ansi.go              # ANSI parsing, RichText segments
│   │   ├── config.go            # SysManConfig loading/saving
│   │   ├── filter.go            # Generic Filter[T]
│   │   ├── hover_button.go      # HoverableButton
│   │   ├── i18n.go              # Translation loading
│   │   ├── icon.go              # AppIcon, SetWindowIcon, LogoImage
│   │   └── version.go           # Version, AppAuthor, AppLicense, AppURL
│   ├── serman/                  # Services plugin (runit via sv)
│   ├── pkgman/                  # Packages plugin (xbps)
│   ├── srcman/                  # Templates plugin (xbps-src)
│   ├── infman/                  # System info plugin (fastfetch/neofetch)
│   ├── ugsman/                  # Users & Groups plugin
│   ├── vmsman/                  # VM manager plugin (QEMU)
│   ├── cmd/                     # Entry points
│   │   ├── sysman-gui/          # Full system manager GUI
│   │   ├── sysman-tui/          # Full system manager TUI
│   │   ├── serman-gui/          # Services standalone GUI
│   │   ├── serman-tui/          # Services standalone TUI
│   │   ├── ugsman-gui/          # Users & Groups GUI
│   │   ├── ugsman-tui/          # Users & Groups TUI
│   │   ├── infman-gui/          # System info GUI
│   │   ├── infman-tui/          # System info TUI
│   │   ├── srcman-gui/          # Templates GUI
│   │   ├── srcman-tui/          # Templates TUI
│   │   ├── pkgman-gui/          # Packages GUI
│   │   ├── pkgman-tui/          # Packages TUI
│   │   ├── vmsman-gui/          # VM manager GUI
│   │   └── vmsman-tui/          # VM manager TUI
│   └── lang/                    # Translation files per module
│       ├── serman/{en,cs}.yaml
│       ├── pkgman/{en,cs}.yaml
│       ├── srcman/{en,cs}.yaml
│       ├── infman/{en,cs}.yaml
│       ├── ugsman/{en,cs}.yaml
│       └── vmsman/{en,cs}.yaml
└── web/                         # Static website (50+ themes)
```

## Module Naming Convention

| Old name | New name | Description |
|---|---|---|
| `plugin` | `serman` | Services (runit) |
| `xbps-pkg` | `pkgman` | Packages (xbps) |
| `xbps-src` | `srcman` | Templates (xbps-src) |
| `sysinfo` | `infman` | System info (fastfetch) |
| `usergroups` | `ugsman` | Users & Groups |
| `vmman` | `vmsman` | VM manager (QEMU) |

## Build Commands

```bash
make build              # All GUI binaries (with PIE)
make build-tui          # All TUI binaries
make build-sysman       # Single binary
make build-sysman-tui   # Single TUI binary
make build-plugins      # Dynamic .so plugins
make test               # Tests with race detector
make lint               # go vet
make fmt                # gofmt -s
make clean              # Remove build/
```

## Plugin API

Every plugin implements `api.PluginIF`:

```go
type PluginIF interface {
    Name() string
    Content(win fyne.Window) fyne.CanvasObject  // GUI
    Model() tea.Model                            // TUI
}
```

## Dependencies

```
fyne.io/fyne/v2                    GUI framework (CGO required)
github.com/charmbracelet/bubbletea TUI framework
github.com/charmbracelet/lipgloss  Terminal styling
github.com/charmbracelet/bubbles   TUI components
github.com/creack/pty              PTY for ANSI color support
gopkg.in/yaml.v3                   YAML parsing
golang.org/x/term                  Terminal detection
```

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `SERVICEDIR` | runit service definitions | `/etc/sv` |
| `SERVICEDESTDIR` | enabled services directory | `/var/service` |
| `SYSMAN_LANG` | language override (`cs`, `en`) | auto from `LANG` |
| `XBPS_DISTDIR` | path to void-packages clone | `~/void` |
| `PLUGIN_DIR` | directory for dynamic `.so` plugins | `./plugins` |
| `VMDIR` | VM directory | `~/vm` |
