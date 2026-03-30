# System Manager

**System Manager** is a modular desktop and terminal application for Void Linux that combines service management, package management, and system information into a single tabbed interface.

It is also a plugin framework — each tab is an independently embeddable component that can be used standalone or composed into any Fyne or Bubbletea application.

---

## Tabs (built-in plugins)

| Tab | Plugin | Backend |
|---|---|---|
| **SysInfo** | `sysinfo` | fastfetch |
| **Packages** | `xbps-pkg` | xbps (`xbps-query`, `xbps-install`) |
| **Templates** | `xbps-src` | xbps-src void-packages |
| **Services** | `plugin` (svman) | runit (`sv`, `sudo ln/rm`) |

---

## Quick start

```bash
git clone https://codeberg.org/oSoWoSo/svman
cd svman
make build-sysmanager   # builds build/sysmanager
./build/sysmanager      # GUI (auto-detects display)
./build/sysmanager --tui
```

### Standalone binaries

Each plugin can also run independently:

```bash
make build          # build/svman       — Services only (GUI+TUI)
make build-tui      # build/svman-tui   — Services only (TUI, CGO-free)
make build-xbps-pkg # build/xbps-pkg    — Packages only
make build-xbps-src # build/xbps-src    — Templates only
make build-sysinfo  # build/sysinfo     — SysInfo only
```

### CGO dependencies (required for GUI)

```bash
# Void Linux
sudo xbps-install gcc pkg-config libX11-devel libXrandr-devel libXinerama-devel libXcursor-devel libXi-devel mesa-devel
# Debian / Ubuntu
sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev
```

---

## Usage

```bash
sysmanager              # GUI (default when display available)
sysmanager --tui        # TUI
sysmanager --help

svman                   # Services GUI
svman --tui             # Services TUI
```

### Environment variables

| Variable | Description | Default |
|---|---|---|
| `SERVICEDIR` | runit service definitions | `/etc/sv` |
| `SERVICEDESTDIR` | enabled services directory | `/var/service` |
| `SVMAN_LANG` | language override (`cs`, `en`) | auto from `LANG` |
| `PLUGIN_DIR` | directory for dynamic `.so` plugins | `./plugins` |

---

## Controls

### TUI — Services tab

| Key | Action |
|---|---|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` / `Space` | Enable / disable |
| `s` | Start |
| `x` | Stop |
| `t` | Restart |
| `l` | Reload (HUP) |
| `p` | Pause (SIGSTOP) |
| `c` | Continue (SIGCONT) |
| `K` | Kill (SIGKILL) |
| `/` | Search |
| `Tab` | Cycle filter (All / Enabled / Disabled) |
| `r` | Reload list |
| `q` / `Esc` | Quit |

### TUI — tab switching (sysmanager)

| Key | Action |
|---|---|
| `1` | SysInfo |
| `2` | Packages |
| `3` | Templates |
| `4` | Services |
| `Ctrl+C` | Quit |

---

## Project structure

```
svman/
├── main.go                    # svman entry point (Services standalone)
├── Makefile
├── lang/                      # Translation files (cs, en)
├── api/
│   └── plugin.go              # PluginIF interface
├── plugin/                    # Services plugin (runit via sv)
│   ├── plugin.go              # Plugin, New, NewRunit, NewWithBackend
│   ├── plugin_gui.go          # Content / ShowAbout
│   ├── gui.go                 # Fyne GUI
│   ├── tui.go                 # Bubbletea TUI
│   ├── common.go              # Service, Backend interface, RunitBackend
│   └── i18n.go                # Translations
├── xbps-pkg/                  # Packages plugin (xbps)
│   ├── plugin.go              # Plugin, New, NewXbps, NewWithBackend
│   ├── common.go              # PkgBackend interface, XbpsBackend
│   ├── plugin_gui.go          # Content
│   └── tui.go                 # Bubbletea TUI
├── xbps-src/                  # Templates plugin (xbps-src)
├── sysinfo/                   # SysInfo plugin (fastfetch)
├── cmd/
│   ├── sysmanager/            # System Manager (all plugins, GUI+TUI)
│   ├── svman-tui/             # Services TUI-only binary (CGO-free)
│   ├── xbps-pkg/              # Packages standalone binary
│   ├── xbps-src/              # Templates standalone binary
│   └── sysinfo/               # SysInfo standalone binary
└── pluginentry/               # Dynamic .so entry points
    ├── svman/
    ├── xbps-pkg/
    ├── xbps-src/
    └── sysinfo/
```

---

## Plugin API

Every plugin implements `api.PluginIF`:

```go
type PluginIF interface {
    Name() string
    Content(win fyne.Window) fyne.CanvasObject  // GUI
    Model() tea.Model                            // TUI
}
```

### Embedding Services in a Fyne application

```go
import svman "codeberg.org/oSoWoSo/svman/plugin"

svman.InitI18n()
p := svman.New("/etc/sv", "/var/service")   // runit backend
// or with a custom backend:
p = svman.NewWithBackend(&MyOpenRCBackend{})

tabs := container.NewAppTabs(
    container.NewTabItem(p.Name(), p.Content(win)),
)
```

### Embedding Packages in a Fyne application

```go
import xbpspkg "codeberg.org/oSoWoSo/svman/xbps-pkg"

p := xbpspkg.New()                         // xbps backend
// or with a custom backend:
p = xbpspkg.NewWithBackend(&MyAptBackend{})
```

### Custom backend (Services)

```go
type MyOpenRCBackend struct{}

func (b *MyOpenRCBackend) Dirs() (string, string)         { return "/etc/init.d", "/etc/runlevels/default" }
func (b *MyOpenRCBackend) List() []plugin.Service         { … }
func (b *MyOpenRCBackend) Enable(name string) error       { … }
func (b *MyOpenRCBackend) Disable(name string) error      { … }
func (b *MyOpenRCBackend) Status(name string) plugin.ServiceStatus { … }
func (b *MyOpenRCBackend) Start(name string) error        { … }
func (b *MyOpenRCBackend) Stop(name string) error         { … }
func (b *MyOpenRCBackend) Restart(name string) error      { … }
func (b *MyOpenRCBackend) Reload(name string) error       { … }
func (b *MyOpenRCBackend) Pause(name string) error        { … }
func (b *MyOpenRCBackend) Continue(name string) error     { … }
func (b *MyOpenRCBackend) Kill(name string) error         { … }

p := svman.NewWithBackend(&MyOpenRCBackend{})
```

### Dynamic plugin loading

```bash
make build-plugins              # build/plugins/*.so
PLUGIN_DIR=./build/plugins ./build/sysmanager
```

Custom `.so` plugin:

```go
// myplugin/main.go
package main

import "codeberg.org/oSoWoSo/svman/api"

func New() api.PluginIF { return &myPlugin{} }
```

```bash
go build -buildmode=plugin -o plugins/myplugin.so ./myplugin/
```

> **Note:** Go plugins require the same Go version and module dependencies as the host binary. Dynamic loading is Linux-only.

---

## Makefile targets

| Target | Output |
|---|---|
| `make build` | `build/svman` — Services standalone (GUI+TUI) |
| `make build-all` | all standalone binaries at once |
| `make build-tui` | `build/svman-tui` — Services TUI (CGO-free) |
| `make build-sysmanager` | `build/sysmanager` — full system manager |
| `make build-xbps-pkg` | `build/xbps-pkg` — Packages standalone |
| `make build-xbps-src` | `build/xbps-src` — Templates standalone |
| `make build-sysinfo` | `build/sysinfo` — SysInfo standalone |
| `make build-plugins` | `build/plugins/*.so` — dynamic plugins |
| `make test` | run tests with race detector |
| `make lint` | `go vet` + golangci-lint |
| `make fmt` | `gofmt` |
| `make release` | release binary + sha256 + tar.gz |
| `make clean` | remove `build/` |

---

## Security

The Services plugin uses `sudo` for symlink operations in `/var/service` and `sv` commands.
Optional passwordless sudo rules (add via `visudo`):

```sudoers
%wheel ALL=(ALL) NOPASSWD: /usr/bin/ln -s /etc/sv/* /var/service/*
%wheel ALL=(ALL) NOPASSWD: /usr/bin/rm /var/service/*
%wheel ALL=(ALL) NOPASSWD: /usr/bin/sv * /var/service/*
```

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

## License

MIT — see [LICENSE](LICENSE)

## Author

[oSoWoSo](https://codeberg.org/oSoWoSo)
