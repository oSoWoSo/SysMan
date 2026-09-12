# SysMan — System Manager
(Wibe coded proof of concept)

**SysMan** is a modular desktop and terminal application for Void Linux that combines service management, package management, template management, system information, and user/group management into a single tabbed interface.

It is also a plugin framework — each tab is an independently embeddable component that can be used standalone or composed into any Fyne or Bubbletea application.

---

## Tabs (built-in plugins)

| Tab | Plugin | Backend |
|---|---|---|
| **SysInfo** | `infman` | fastfetch |
| **Packages** | `pkgman` | xbps (`xbps-query`, `xbps-install`) |
| **Templates** | `srcman` | xbps-src void-packages |
| **Services** | `serman` | runit (`sv`, `pkexec`/`doas`/`sudo`) — system + user services (managed without elevation) |
| **Users & Groups** | `ugsman` | `/etc/passwd`, `/etc/group` |

---

## Quick start

```bash
git clone https://codeberg.org/oSoWoSo/SysMan
cd SysMan
make build-sysman   # builds build/sysman
./build/sysman      # GUI (auto-detects display)
./build/sysman --tui
```

### Standalone binaries

Each plugin can also run independently:

| Binary | Description | CGO |
|---|---|---|
| `sysman` | Full system manager (all plugins, GUI + TUI) | required |
| `sysman-tui` | Full system manager (TUI entry) | required |
| `serman` | Services manager (GUI + TUI) | required |
| `serman-tui` | Services manager (TUI only) | free |
| `ugsman` | Users & Groups manager (GUI + TUI) | required |
| `ugsman-tui` | Users & Groups manager (TUI only) | free |
| `infman` | System info (GUI + TUI) | required |
| `infman-tui` | System info (TUI only) | free |
| `srcman` | Templates manager (GUI + TUI) | required |
| `srcman-tui` | Templates manager (TUI only) | free |
| `pkgman` | Package manager (GUI + TUI) | required |
| `pkgman-tui` | Package manager (TUI only) | free |
| `vmsman` | VM manager (GUI + TUI) | required |
| `vmsman-tui` | VM manager (TUI only) | free |

```bash
make build          # all GUI binaries
make build-serman   # build/serman only
make build-serman-tui
# … see Makefile targets below
```

### CGO dependencies (required for GUI builds)

```bash
# Void Linux
sudo xbps-install go gcc pkg-config libXxf86vm-devel libX11-devel libXrandr-devel libXinerama-devel libXcursor-devel libXi-devel MesaLib-devel wayland-devel libxkbcommon-devel
```

### Runtime dependencies
```bash
libGL libX11 libXcursor libXi libXinerama libXrandr libXxf86vm
```

---

## Usage

```bash
sysman              # GUI (default when display available)
sysman --tui        # TUI
sysman --help

serman               # Services GUI
serman --tui         # Services TUI

ugsman               # Users & Groups GUI
ugsman-tui           # Users & Groups TUI only

infman               # SysInfo GUI
infman-tui           # SysInfo TUI only

srcman              # Templates GUI (reads $XBPS_DISTDIR)
srcman-tui          # Templates TUI only

pkgman              # Packages GUI
pkgman-tui          # Packages TUI only
```

### Environment variables

| Variable | Description | Default |
|---|---|---|
| `SERVICEDIR` | runit service definitions | `/etc/sv` |
| `SERVICEDESTDIR` | enabled services directory | `/var/service` |
| `USER_SERVICEDIR` | user runit service definitions | `~/.config/service` |
| `USER_SERVICEDESTDIR` | user enabled services directory | `~/service` |
| `SYSMAN_LANG` | language override (`cs`, `en`) | auto from `LANG` |
| `XBPS_DISTDIR` | path to void-packages clone | `~/void` |
| `PLUGIN_DIR` | directory for dynamic `.so` plugins | `./plugins` |
| `VMDIR` | VM directory | `~/vm` |

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
| `z` | Switch between system / user services (when multi-scope) |
| `r` | Reload list |
| `q` / `Esc` | Quit |

### TUI — Users & Groups tab

| Key | Action |
|---|---|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `1` | Users tab |
| `2` | Groups tab |
| `s` | Toggle system users |
| `r` | Refresh |
| `q` / `Esc` | Quit |

### TUI — tab switching (sysman)

| Key | Action |
|---|---|
| `1` | SysInfo |
| `2` | Packages |
| `3` | Templates |
| `4` | Services |
| `5` | Users & Groups |
| `Ctrl+C` | Quit |

### GUI — all standalone windows

| Key | Action |
|---|---|
| `Esc` | Quit |

---

## Project structure

```
SysMan/
├── main.go                    # sysman entry point
├── Makefile
├── lang/                      # Translation files (cs, en)
├── api/
│   └── plugin.go              # PluginIF interface
├── serman/                    # Services plugin (runit via sv)
│   ├── plugin.go              # Plugin, New, NewRunit, NewWithBackend
│   ├── plugin_gui.go          # Content / ShowAbout
│   ├── gui.go                 # Fyne GUI
│   ├── tui.go                 # Bubbletea TUI
│   ├── common.go              # Service, Backend interface, RunitBackend
│   └── i18n.go                # Translations
├── pkgman/                    # Packages plugin (xbps)
│   ├── plugin.go
│   ├── common.go              # PkgBackend interface, XbpsBackend
│   ├── plugin_gui.go          # Content
│   └── tui.go                 # Bubbletea TUI
├── srcman/                    # Templates plugin
├── infman/                    # SysInfo plugin (fastfetch)
├── ugsman/                    # Users & Groups plugin
│   ├── plugin.go
│   ├── gui.go                 # Fyne GUI + RunGUI
│   ├── tui.go                 # Bubbletea TUI + RunTUI
│   └── users.go               # User/Group loading
├── cmd/
│   ├── sysman-gui/            # sysman / sysman-tui
│   ├── serman-gui/            # serman
│   ├── serman-tui/            # serman-tui (CGO-free)
│   ├── ugsman-gui/            # ugsman
│   ├── ugsman-tui/            # ugsman-tui (CGO-free)
│   ├── pkgman-gui/            # pkgman
│   ├── pkgman-tui/            # pkgman-tui (CGO-free)
│   ├── infman-gui/            # infman
│   ├── infman-tui/            # infman-tui (CGO-free)
│   ├── srcman-gui/            # srcman
│   ├── srcman-tui/            # srcman-tui (CGO-free)
│   ├── vmsman-gui/            # vmsman
│   └── vmsman-tui/            # vmsman-tui (CGO-free)
└── vmsman/                    # VM manager plugin
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
import serman "codeberg.org/oSoWoSo/SysMan/serman"

serman.InitI18n()
p := serman.New("/etc/sv", "/var/service")   // runit backend
// or with a custom backend:
p = serman.NewWithBackend(&MyOpenRCBackend{})

tabs := container.NewAppTabs(
    container.NewTabItem(p.Name(), p.Content(win)),
)
```

### Embedding Packages in a Fyne application

```go
import pkgman "codeberg.org/oSoWoSo/SysMan/pkgman"

p := pkgman.New()                         // xbps backend
// or with a custom backend:
p = pkgman.NewWithBackend(&MyAptBackend{})
```

### Embedding Users & Groups in a Fyne application

```go
import "codeberg.org/oSoWoSo/SysMan/ugsman"

p := ugsman.New()
tabs := container.NewAppTabs(
    container.NewTabItem(p.Name(), p.Content(win)),
)
// TUI:
_ = p.Model()
```

### Custom backend (Services)

```go
type MyOpenRCBackend struct{}

func (b *MyOpenRCBackend) Dirs() (string, string)                         { return "/etc/init.d", "/etc/runlevels/default" }
func (b *MyOpenRCBackend) List() []serman.Service                         { … }
func (b *MyOpenRCBackend) Enable(name string) error                       { … }
func (b *MyOpenRCBackend) Disable(name string) error                      { … }
func (b *MyOpenRCBackend) Status(name string) serman.ServiceStatus        { … }
func (b *MyOpenRCBackend) StatusAll(names []string) map[string]serman.ServiceStatus { … }
func (b *MyOpenRCBackend) Start(name string) error                        { … }
func (b *MyOpenRCBackend) Stop(name string) error                         { … }
func (b *MyOpenRCBackend) Restart(name string) error                      { … }
func (b *MyOpenRCBackend) Reload(name string) error                       { … }
func (b *MyOpenRCBackend) Pause(name string) error                        { … }
func (b *MyOpenRCBackend) Continue(name string) error                     { … }
func (b *MyOpenRCBackend) Kill(name string) error                         { … }

p := serman.NewWithBackend(&MyOpenRCBackend{})
```

### Dynamic plugin loading

```bash
make build-plugins              # build/plugins/*.so
PLUGIN_DIR=./build/plugins ./build/sysman
```

Custom `.so` plugin:

```go
// myplugin/main.go
package main

import "codeberg.org/oSoWoSo/SysMan/api"

func New() api.PluginIF { return &myPlugin{} }
```

```bash
go build -buildmode=plugin -o plugins/myplugin.so ./myplugin/
```

> **Note:** Go plugins require the same Go version and module dependencies as the host binary.

---

## Makefile targets

| Target | Output |
|---|---|
| `make build` | all 14 binaries |
| `make build-sysman` | `build/sysman` — full system manager |
| `make build-sysman-tui` | `build/sysman-tui` — full system manager (TUI entry) |
| `make build-serman` | `build/serman` — Services standalone |
| `make build-serman-tui` | `build/serman-tui` — Services TUI (CGO-free) |
| `make build-ugsman` | `build/ugsman` — Users & Groups standalone |
| `make build-ugsman-tui` | `build/ugsman-tui` — Users & Groups TUI (CGO-free) |
| `make build-infman` | `build/infman` — SysInfo standalone |
| `make build-infman-tui` | `build/infman-tui` — SysInfo TUI (CGO-free) |
| `make build-srcman` | `build/srcman` — Templates standalone |
| `make build-srcman-tui` | `build/srcman-tui` — Templates TUI (CGO-free) |
| `make build-pkgman` | `build/pkgman` — Packages standalone |
| `make build-pkgman-tui` | `build/pkgman-tui` — Packages TUI (CGO-free) |
| `make build-vmsman` | `build/vmsman` — VM manager standalone |
| `make build-vmsman-tui` | `build/vmsman-tui` — VM manager TUI (CGO-free) |
| `make build-plugins` | `build/plugins/*.so` — dynamic plugins |
| `make test` | run tests with race detector |
| `make lint` | `go vet` + golangci-lint |
| `make fmt` | `gofmt -s` |
| `make install` | install all binaries + lang files |
| `make uninstall` | remove installed files |
| `make release` | per-binary tarballs with sha256 checksums |
| `make clean` | remove `build/` |

---

## Security

System-scope service operations (`sv`, symlink enable/disable) require elevated privileges and are run via `pkexec`, `doas`, or `sudo` (whichever is available). **User-scope services** are managed without elevation — no `sudo` prompt is shown while the user-only scope view is active.

Elevated privileges are only requested for system-service changes and for an explicit status refresh (Reload button / `r`). Browsing, filtering, searching, and switching between the system/user scope views never prompt for elevation — the service list always displays, while statuses (which need privileges) are fetched only on demand.

The scope shown on first display defaults to system services; set `default_scope: user` in `~/.config/sysman/sysman.conf` (or choose *Default Scope* in the module settings dialog) to start on user services instead. The All/Enabled/Disabled filters always respect the active scope — they only ever list services from the currently selected system or user scope, never both at once.

Optional passwordless rules (add via `visudo` or `/etc/doas.conf`):

```sudoers
# sudo
%wheel ALL=(ALL) NOPASSWD: /usr/bin/ln -s /etc/sv/* /var/service/*
%wheel ALL=(ALL) NOPASSWD: /usr/bin/rm /var/service/*
%wheel ALL=(ALL) NOPASSWD: /usr/bin/sv * /var/service/*
```

```
# doas (/etc/doas.conf)
permit nopass :wheel cmd ln
permit nopass :wheel cmd rm
permit nopass :wheel cmd sv
```

### Per-user services (runit)

SysMan follows the Void Linux per-user service layouts. The user scope is activated when **either** directory exists, and it is always managed **without** elevation:

- **Definitions + live dir** (runit-manager/turnstile style): service run dirs live in `~/.config/service` and are symlinked into the live directory `~/service`. Enabled services are the symlinks; control and status use `sv <action> ~/service/<name>`.
- **Live dir only** (classic `runsvdir-<username>` layout): run directories or symlinks live directly in `~/service`. SysMan lists them directly as enabled services.

Override the defaults with `USER_SERVICEDIR`/`USER_SERVICEDESTDIR` (or `user_service_dir`/`user_service_dest_dir` in `~/.config/sysman/sysman.conf`) — e.g. turnstile users keep definitions in `~/.config/service` by default.

Smoke-test without installing anything:

```sh
mkdir -p ~/service/smoketest
printf '#!/bin/sh\nexec sleep 1000\n' > ~/service/smoketest/run
chmod +x ~/service/smoketest/run
```

`smoketest` then appears in the user scope with no elevation prompt. Without a supervising `runsvdir-<user>` the status reports "not running", which is fine for verifying the list, scope toggle, and control calls.

### Packages (pkgman)

Packages offer two settings (gear button in the toolbar):

- **AppImage dir** — directory scanned for installed apps. Leave empty to inherit the path from the appman/am configuration (`~/.config/appman/appman-config` or `~/.config/AM/appman-config`, falling back to `~/Applications`); an explicit value overrides it. Installed-state detection follows the AM layout: each installed app is a subdirectory containing a `remove` script (the AppImage binary has no `.AppImage` suffix), or a lone `.AppImage` file (`--launcher` style). The same scan covers the system-wide am directory `/opt`.
- **Custom xbps repositories** — a user-level pool of repository URLs stored in `~/.config/sysman/sysman.conf`. Adding/removing entries needs no privileges. Once a repository is **used**, the active set is written elevated to `/etc/xbps.d/<repos_file>` (default `sysman-repos.conf`) as `repository=<url>` lines — this is the only place the managed file is touched, never Void's own `00-repository-main.conf`. A later package sync may ask you to confirm the repository's signing key.

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

AGPL-3.0-only — see [LICENSE](LICENSE)

## Author

zenobit @ [oSoWoSo](https://oSoWoSo.org)

### Source

[Codeberg](https://codeberg.org/oSoWoSo/SysMan)

### Mirrors

[GitHub](https://github.com/oSoWoSo/SysMan)
