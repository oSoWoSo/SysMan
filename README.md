# svman – runit Service Manager

**svman** is a service manager for systems running the **runit** init system.
It provides a graphical (GUI) and terminal (TUI) interface for managing symlinks in `/var/service`.

svman is also an **embeddable plugin** — it can be dropped into any Fyne or Bubbletea application as a tab or panel without modifying its source code.

---

## Features

- Load and display services from the runit directory
- Enable / disable services via `sudo`
- Filter by state (All / Enabled / Disabled)
- Real-time search
- Detail panel for the selected service
- About dialog with version info
- Czech and English interface (auto-detected from `LANG`)
- Adaptive colors (light / dark terminal)
- Embeddable plugin API for system managers
- TUI-only binary for cross-platform use (no CGO / no Fyne)

---

## Quick start

```bash
git clone https://codeberg.org/oSoWoSo/svman
cd svman
make build          # builds build/svman
./build/svman       # launches GUI (default)
./build/svman --tui # launches TUI
```

### System installation

```bash
sudo cp build/svman /usr/local/bin/
sudo mkdir -p /usr/local/share/svman/lang
sudo cp lang/*.yaml /usr/local/share/svman/lang/
```

### CGO dependencies (required for GUI build)

```bash
# Debian / Ubuntu / Void
sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev
# Arch
sudo pacman -S gcc mesa libxcursor libxrandr libxinerama libxi
```

---

## Usage

```bash
svman           # GUI (default)
svman --gui     # GUI explicitly
svman --tui     # TUI (terminal)
svman --help    # help
```

### Environment variables

| Variable | Description | Default |
|---|---|---|
| `SERVICEDIR` | Service definition directory | `/etc/sv` |
| `SERVICEDESTDIR` | Enabled services symlink directory | `/var/service` |
| `SVMAN_LANG` | Language override (`cs`, `en`) | auto from `LANG` |

```bash
SERVICEDIR=/home/user/sv SVMAN_LANG=en svman --tui
```

---

## Controls

### TUI

| Key | Action |
|---|---|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` / `Space` | Enable / disable service |
| `/` | Search |
| `Esc` | Clear search / cancel |
| `Tab` | Cycle filters (All → Enabled → Disabled) |
| `r` | Reload service list |
| `q` / `Ctrl+C` / `Esc` | Quit |

### GUI

- **Click** on a service row to enable / disable it
- **ⓘ** button (bottom-left) opens the About dialog
- **Refresh** button reloads the list

---

## Build modes

| Binary | CGO | Fyne | Platforms |
|---|---|---|---|
| `build/svman` | required | ✓ | linux/amd64 |
| `build/svman-tui` | not needed | ✗ | any |

```bash
make build          # GUI+TUI binary (requires CGO)
make build-tui      # TUI-only binary, cross-compilable

# Cross-compile TUI for arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -tags tui_only -o svman-arm64 ./cmd/svman-tui/
```

---

## Plugin API

svman exposes an embeddable `plugin.Plugin` type that implements `api.PluginIF`:

```go
import svman "codeberg.org/oSoWoSo/svman/plugin"

p := svman.New(serviceDir, serviceDestDir)
p.Name()          // "Services"
p.Content(win)    // fyne.CanvasObject — embed in any container
p.Model()         // tea.Model — wrap in your own tea.Program
p.ShowAbout(win)  // show About dialog
```

### Embedding in a Fyne application

```go
svman.InitI18n()
p := svman.New("/etc/sv", "/var/service")
tabs := container.NewAppTabs(
    container.NewTabItem(p.Name(), p.Content(win)),
)
```

### Embedding in a Bubbletea application

```go
svman.InitI18n()
p := svman.New("/etc/sv", "/var/service")
program := tea.NewProgram(p.Model(), tea.WithAltScreen())
```

---

## System Manager demo

`cmd/sysmanager` is a demo application that embeds multiple plugins in one window.

```bash
make build-sysmanager
./build/sysmanager          # GUI with tabs
./build/sysmanager --tui    # TUI with F1/F2 tab switching
```

Built-in plugins: **Services** (svman) + **System Info** (testplugin).

### Adding plugins without rebuilding

Plugins are loaded at runtime from `$PLUGIN_DIR` (default: `./plugins/`).
Each `.so` file must export `func New() api.PluginIF`.

```bash
# Build .so plugins
make build-plugins

# Run system manager with dynamic plugins
PLUGIN_DIR=./build/plugins ./build/sysmanager
```

Build your own plugin:
```go
// myplugin/main.go (compiled with -buildmode=plugin)
package main

import "codeberg.org/oSoWoSo/svman/api"

func New() api.PluginIF { return &myPlugin{} }

type myPlugin struct{}
func (p *myPlugin) Name() string                              { return "My Plugin" }
func (p *myPlugin) Content(win fyne.Window) fyne.CanvasObject { ... }
func (p *myPlugin) Model() tea.Model                          { ... }
```

```bash
go build -buildmode=plugin -o plugins/myplugin.so ./myplugin/
```

> **Note:** Go plugins require matching Go version and dependencies between host and plugin.
> Dynamic loading is supported on Linux only.

---

## Project structure

```
svman/
├── main.go                    # svman entry point (GUI+TUI)
├── Makefile
├── lang/                      # Translation files (cs, en)
├── api/
│   └── plugin.go              # PluginIF interface
├── plugin/                    # Embeddable svman plugin library
│   ├── plugin.go              # Plugin struct, Name / New / Model
│   ├── plugin_gui.go          # Content / ShowAbout  (!tui_only)
│   ├── gui.go                 # Fyne GUI              (!tui_only)
│   ├── tui.go                 # Bubbletea TUI
│   ├── common.go              # Service, LoadServices, Enable/Disable
│   └── i18n.go                # Translations
├── testplugin/                # Demo "System Info" plugin
├── cmd/
│   ├── svman-tui/             # TUI-only binary (CGO-free)
│   ├── sysmanager/            # Demo system manager
│   └── testplugin/            # testplugin standalone binary
└── pluginentry/               # .so entry points for dynamic loading
    ├── svman/
    └── testplugin/
```

---

## Makefile targets

| Target | Description |
|---|---|
| `make build` | Build `build/svman` (GUI+TUI) |
| `make build-sysmanager` | Build `build/sysmanager` |
| `make build-testplugin` | Build `build/testplugin` |
| `make build-plugins` | Build `build/plugins/*.so` |
| `make test` | Run tests with race detector |
| `make lint` | `go vet` + golangci-lint |
| `make fmt` | Format code with `gofmt` |
| `make release` | Release binary + sha256 + tar.gz |
| `make clean` | Remove `build/` |

---

## Dependencies

```
fyne.io/fyne/v2                    GUI framework (CGO required)
github.com/charmbracelet/bubbletea TUI framework
github.com/charmbracelet/lipgloss  Terminal styling
github.com/charmbracelet/bubbles   TUI components
gopkg.in/yaml.v3                   YAML parsing
```

---

## Security

svman uses `sudo` for symlink operations in `/var/service`.
Optional passwordless sudo rules (add via `visudo`):

```sudoers
%wheel ALL=(ALL) NOPASSWD: /usr/bin/ln -s /etc/sv/* /var/service/*
%wheel ALL=(ALL) NOPASSWD: /usr/bin/rm /var/service/*
```

---

## License

MIT — see [LICENSE](LICENSE)

## Author

[oSoWoSo](https://codeberg.org/oSoWoSo)
