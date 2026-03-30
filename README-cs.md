# System Manager

**System Manager** je modulární desktopová a terminálová aplikace pro Void Linux, která sdružuje správu služeb, správu balíčků a systémové informace do jediného okna se záložkami.

Je zároveň pluginovým frameworkem — každá záložka je samostatně použitelná komponenta, kterou lze vložit do libovolné aplikace postavené na Fyne nebo Bubbletea.

---

## Záložky (vestavěné pluginy)

| Záložka | Plugin | Backend |
|---|---|---|
| **SysInfo** | `sysinfo` | fastfetch |
| **Packages** | `xbps-pkg` | xbps (`xbps-query`, `xbps-install`) |
| **Templates** | `xbps-src` | xbps-src void-packages |
| **Services** | `plugin` (svman) | runit (`sv`, `sudo ln/rm`) |

---

## Rychlý start

```bash
git clone https://codeberg.org/oSoWoSo/svman
cd svman
make build-sysmanager   # build/sysmanager
./build/sysmanager      # GUI (detekuje display automaticky)
./build/sysmanager --tui
```

### Samostatné binárky

Každý plugin lze provozovat i samostatně:

```bash
make build          # build/svman       — pouze Services (GUI+TUI)
make build-tui      # build/svman-tui   — pouze Services (TUI, bez CGO)
make build-xbps-pkg # build/xbps-pkg    — pouze Packages
make build-xbps-src # build/xbps-src    — pouze Templates
make build-sysinfo  # build/sysinfo     — pouze SysInfo
```

### Závislosti pro GUI (CGO)

```bash
# Void Linux
sudo xbps-install gcc pkg-config libX11-devel libXrandr-devel libXinerama-devel libXcursor-devel libXi-devel mesa-devel
# Debian / Ubuntu
sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev
```

---

## Použití

```bash
sysmanager              # GUI (výchozí, pokud je dostupný display)
sysmanager --tui        # TUI
sysmanager --help

svman                   # Services GUI (samostatně)
svman --tui             # Services TUI (samostatně)
```

### Proměnné prostředí

| Proměnná | Popis | Výchozí |
|---|---|---|
| `SERVICEDIR` | Adresář definic služeb runit | `/etc/sv` |
| `SERVICEDESTDIR` | Adresář povolených služeb | `/var/service` |
| `SVMAN_LANG` | Jazyk rozhraní (`cs`, `en`) | auto z `LANG` |
| `PLUGIN_DIR` | Adresář pro dynamické `.so` pluginy | `./plugins` |

---

## Ovládání

### TUI — záložka Services

| Klávesa | Akce |
|---|---|
| `↑` / `k` | Nahoru |
| `↓` / `j` | Dolů |
| `Enter` / `Space` | Povolit / zakázat |
| `s` | Spustit |
| `x` | Zastavit |
| `t` | Restartovat |
| `l` | Reload (HUP) |
| `p` | Pozastavit (SIGSTOP) |
| `c` | Pokračovat (SIGCONT) |
| `K` | Zabít (SIGKILL) |
| `/` | Hledat |
| `Tab` | Přepnout filtr (Vše / Povolené / Zakázané) |
| `r` | Obnovit seznam |
| `q` / `Esc` | Ukončit |

### TUI — přepínání záložek (sysmanager)

| Klávesa | Akce |
|---|---|
| `1` | SysInfo |
| `2` | Packages |
| `3` | Templates |
| `4` | Services |
| `Ctrl+C` | Ukončit |

---

## Struktura projektu

```
svman/
├── main.go                    # Vstupní bod svman (samostatné Services)
├── Makefile
├── lang/                      # Překlady (cs, en)
├── api/
│   └── plugin.go              # Rozhraní PluginIF
├── plugin/                    # Plugin Services (runit přes sv)
│   ├── plugin.go              # Plugin, New, NewRunit, NewWithBackend
│   ├── plugin_gui.go          # Content / ShowAbout
│   ├── gui.go                 # Fyne GUI
│   ├── tui.go                 # Bubbletea TUI
│   ├── common.go              # Service, rozhraní Backend, RunitBackend
│   └── i18n.go                # Překlady
├── xbps-pkg/                  # Plugin Packages (xbps)
│   ├── plugin.go              # Plugin, New, NewXbps, NewWithBackend
│   ├── common.go              # Rozhraní PkgBackend, XbpsBackend
│   ├── plugin_gui.go          # Content
│   └── tui.go                 # Bubbletea TUI
├── xbps-src/                  # Plugin Templates (xbps-src)
├── sysinfo/                   # Plugin SysInfo (fastfetch)
├── cmd/
│   ├── sysmanager/            # System Manager (všechny pluginy, GUI+TUI)
│   ├── svman-tui/             # Services TUI bez CGO
│   ├── xbps-pkg/              # Packages samostatně
│   ├── xbps-src/              # Templates samostatně
│   └── sysinfo/               # SysInfo samostatně
└── pluginentry/               # Vstupní body pro dynamické .so pluginy
    ├── svman/
    ├── xbps-pkg/
    ├── xbps-src/
    └── sysinfo/
```

---

## Plugin API

Každý plugin implementuje rozhraní `api.PluginIF`:

```go
type PluginIF interface {
    Name() string
    Content(win fyne.Window) fyne.CanvasObject  // GUI
    Model() tea.Model                            // TUI
}
```

### Vložení Services do Fyne aplikace

```go
import svman "codeberg.org/oSoWoSo/svman/plugin"

svman.InitI18n()
p := svman.New("/etc/sv", "/var/service")   // runit backend
// nebo s vlastním backendem:
p = svman.NewWithBackend(&MyOpenRCBackend{})

tabs := container.NewAppTabs(
    container.NewTabItem(p.Name(), p.Content(win)),
)
```

### Vložení Packages do Fyne aplikace

```go
import xbpspkg "codeberg.org/oSoWoSo/svman/xbps-pkg"

p := xbpspkg.New()                         // xbps backend
// nebo s vlastním backendem:
p = xbpspkg.NewWithBackend(&MyAptBackend{})
```

### Vlastní backend (Services)

```go
type MyOpenRCBackend struct{}

func (b *MyOpenRCBackend) Dirs() (string, string)              { return "/etc/init.d", "/etc/runlevels/default" }
func (b *MyOpenRCBackend) List() []plugin.Service              { … }
func (b *MyOpenRCBackend) Enable(name string) error            { … }
func (b *MyOpenRCBackend) Disable(name string) error           { … }
func (b *MyOpenRCBackend) Status(name string) plugin.ServiceStatus { … }
func (b *MyOpenRCBackend) Start(name string) error             { … }
func (b *MyOpenRCBackend) Stop(name string) error              { … }
func (b *MyOpenRCBackend) Restart(name string) error           { … }
func (b *MyOpenRCBackend) Reload(name string) error            { … }
func (b *MyOpenRCBackend) Pause(name string) error             { … }
func (b *MyOpenRCBackend) Continue(name string) error          { … }
func (b *MyOpenRCBackend) Kill(name string) error              { … }

p := svman.NewWithBackend(&MyOpenRCBackend{})
```

### Dynamické načítání pluginů

```bash
make build-plugins              # build/plugins/*.so
PLUGIN_DIR=./build/plugins ./build/sysmanager
```

Vlastní `.so` plugin:

```go
// myplugin/main.go
package main

import "codeberg.org/oSoWoSo/svman/api"

func New() api.PluginIF { return &myPlugin{} }
```

```bash
go build -buildmode=plugin -o plugins/myplugin.so ./myplugin/
```

> **Poznámka:** Go pluginy vyžadují shodnou verzi Go a závislostí s hostitelským programem. Dynamické načítání funguje pouze na Linuxu.

---

## Makefile cíle

| Cíl | Výstup |
|---|---|
| `make build` | `build/svman` — Services samostatně (GUI+TUI) |
| `make build-all` | všechny samostatné binárky najednou |
| `make build-tui` | `build/svman-tui` — Services TUI (bez CGO) |
| `make build-sysmanager` | `build/sysmanager` — kompletní system manager |
| `make build-xbps-pkg` | `build/xbps-pkg` — Packages samostatně |
| `make build-xbps-src` | `build/xbps-src` — Templates samostatně |
| `make build-sysinfo` | `build/sysinfo` — SysInfo samostatně |
| `make build-plugins` | `build/plugins/*.so` — dynamické pluginy |
| `make test` | testy s detektorem závodů |
| `make lint` | `go vet` + golangci-lint |
| `make fmt` | `gofmt` |
| `make release` | release binárka + sha256 + tar.gz |
| `make clean` | smaže `build/` |

---

## Bezpečnost

Plugin Services používá `sudo` pro operace se symlinky v `/var/service` a příkazy `sv`.
Volitelná konfigurace bezheslového sudo (přidejte přes `visudo`):

```sudoers
%wheel ALL=(ALL) NOPASSWD: /usr/bin/ln -s /etc/sv/* /var/service/*
%wheel ALL=(ALL) NOPASSWD: /usr/bin/rm /var/service/*
%wheel ALL=(ALL) NOPASSWD: /usr/bin/sv * /var/service/*
```

---

## Závislosti

```
fyne.io/fyne/v2                    GUI framework (vyžaduje CGO)
github.com/charmbracelet/bubbletea TUI framework
github.com/charmbracelet/lipgloss  Stylování terminálu
github.com/charmbracelet/bubbles   TUI komponenty
gopkg.in/yaml.v3                   Parsování YAML
```

---

## Licence

MIT — viz [LICENSE](LICENSE)

## Autor

[oSoWoSo](https://codeberg.org/oSoWoSo)
