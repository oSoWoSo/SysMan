# SysMan — System Manager

**SysMan** je modulární desktopová a terminálová aplikace pro Void Linux, která sdružuje správu služeb, správu balíčků, správu šablon, systémové informace a správu uživatelů/skupin do jediného okna se záložkami.

Je zároveň pluginovým frameworkem — každá záložka je samostatně použitelná komponenta, kterou lze vložit do libovolné aplikace postavené na Fyne nebo Bubbletea.

---

## Záložky (vestavěné pluginy)

| Záložka | Plugin | Backend |
|---|---|---|
| **SysInfo** | `infman` | fastfetch |
| **Packages** | `pkgman` | xbps (`xbps-query`, `xbps-install`) |
| **Templates** | `srcman` | xbps-src void-packages |
| **Services** | `serman` | runit (`sv`, `pkexec`/`doas`/`sudo`) |
| **Users & Groups** | `ugsman` | `/etc/passwd`, `/etc/group` |

---

## Rychlý start

```bash
git clone https://codeberg.org/oSoWoSo/SysMan
cd SysMan
make build-sysman   # build/sysman
./build/sysman      # GUI (detekuje display automaticky)
./build/sysman --tui
```

### Samostatné binárky

Každý plugin lze provozovat i samostatně:

| Binárka | Popis | CGO |
|---|---|---|
| `sysman` | Kompletní system manager (všechny pluginy, GUI + TUI) | vyžadováno |
| `sysman-tui` | Kompletní system manager (TUI vstup) | vyžadováno |
| `serman` | Správa služeb (GUI + TUI) | vyžadováno |
| `serman-tui` | Správa služeb (pouze TUI) | ne |
| `ugsman` | Správa uživatelů a skupin (GUI + TUI) | vyžadováno |
| `ugsman-tui` | Správa uživatelů a skupin (pouze TUI) | ne |
| `infman` | Systémové informace (GUI + TUI) | vyžadováno |
| `infman-tui` | Systémové informace (pouze TUI) | ne |
| `srcman` | Správa šablon (GUI + TUI) | vyžadováno |
| `srcman-tui` | Správa šablon (pouze TUI) | ne |
| `pkgman` | Správa balíčků (GUI + TUI) | vyžadováno |
| `pkgman-tui` | Správa balíčků (pouze TUI) | ne |
| `vmsman` | Správa VM (GUI + TUI) | vyžadováno |
| `vmsman-tui` | Správa VM (pouze TUI) | ne |

```bash
make build          # všechny 14 binárky
make build-serman   # build/serman
make build-serman-tui
# … viz cíle Makefile níže
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
sysman              # GUI (výchozí, pokud je dostupný display)
sysman --tui        # TUI
sysman --help

serman               # Services GUI
serman --tui         # Services TUI

ugsman               # Users & Groups GUI
ugsman-tui           # Users & Groups (pouze TUI)

infman               # SysInfo GUI
infman-tui           # SysInfo (pouze TUI)

srcman              # Templates GUI (čte $XBPS_DISTDIR)
srcman-tui          # Templates (pouze TUI)

pkgman              # Packages GUI
pkgman-tui          # Packages (pouze TUI)
```

### Proměnné prostředí

| Proměnná | Popis | Výchozí |
|---|---|---|
| `SERVICEDIR` | Adresář definic služeb runit | `/etc/sv` |
| `SERVICEDESTDIR` | Adresář povolených služeb | `/var/service` |
| `SVMAN_LANG` | Jazyk rozhraní (`cs`, `en`) | auto z `LANG` |
| `XBPS_DISTDIR` | Cesta ke klonu void-packages | — |
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

### TUI — záložka Users & Groups

| Klávesa | Akce |
|---|---|
| `↑` / `k` | Nahoru |
| `↓` / `j` | Dolů |
| `1` | Záložka Uživatelé |
| `2` | Záložka Skupiny |
| `s` | Přepnout systémové uživatele |
| `r` | Obnovit |
| `q` / `Esc` | Ukončit |

### TUI — přepínání záložek (sysman)

| Klávesa | Akce |
|---|---|
| `1` | SysInfo |
| `2` | Packages |
| `3` | Templates |
| `4` | Services |
| `5` | Users & Groups |
| `Ctrl+C` | Ukončit |

### GUI — všechna samostatná okna

| Klávesa | Akce |
|---|---|
| `Esc` | Ukončit |

---

## Struktura projektu

```
SysMan/
├── main.go                    # Vstupní bod sysman
├── Makefile
├── lang/                      # Překlady (cs, en)
├── api/
│   └── plugin.go              # Rozhraní PluginIF
├── serman/                    # Plugin Services (runit přes sv)
│   ├── plugin.go              # Plugin, New, NewRunit, NewWithBackend
│   ├── plugin_gui.go          # Content / ShowAbout
│   ├── gui.go                 # Fyne GUI
│   ├── tui.go                 # Bubbletea TUI
│   ├── common.go              # Service, rozhraní Backend, RunitBackend
│   └── i18n.go                # Překlady
├── pkgman/                    # Plugin Packages (xbps)
│   ├── plugin.go
│   ├── common.go              # Rozhraní PkgBackend, XbpsBackend
│   ├── plugin_gui.go          # Content
│   └── tui.go                 # Bubbletea TUI
├── srcman/                    # Plugin Templates
├── infman/                    # Plugin SysInfo (fastfetch)
├── ugsman/                    # Plugin Users & Groups
│   ├── plugin.go
│   ├── gui.go                 # Fyne GUI + RunGUI
│   ├── tui.go                 # Bubbletea TUI + RunTUI
│   └── users.go               # Načítání uživatelů/skupin
├── cmd/
│   ├── sysman-gui/            # sysman / sysman-tui
│   ├── serman-gui/            # serman
│   ├── serman-tui/            # serman-tui (bez CGO)
│   ├── ugsman-gui/            # ugsman
│   ├── ugsman-tui/            # ugsman-tui (bez CGO)
│   ├── pkgman-gui/            # pkgman
│   ├── pkgman-tui/            # pkgman-tui (bez CGO)
│   ├── infman-gui/            # infman
│   ├── infman-tui/            # infman-tui (bez CGO)
│   ├── srcman-gui/            # srcman
│   ├── srcman-tui/            # srcman-tui (bez CGO)
│   ├── vmsman-gui/            # vmsman
│   └── vmsman-tui/            # vmsman-tui (bez CGO)
└── vmsman/                    # Plugin VM manager
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
import serman "codeberg.org/oSoWoSo/SysMan/serman"

serman.InitI18n()
p := serman.New("/etc/sv", "/var/service")   // runit backend
// nebo s vlastním backendem:
p = serman.NewWithBackend(&MyOpenRCBackend{})

tabs := container.NewAppTabs(
    container.NewTabItem(p.Name(), p.Content(win)),
)
```

### Vložení Packages do Fyne aplikace

```go
import pkgman "codeberg.org/oSoWoSo/SysMan/pkgman"

p := pkgman.New()                         // xbps backend
// nebo s vlastním backendem:
p = pkgman.NewWithBackend(&MyAptBackend{})
```

### Vložení Users & Groups do Fyne aplikace

```go
import "codeberg.org/oSoWoSo/SysMan/ugsman"

p := ugsman.New()
tabs := container.NewAppTabs(
    container.NewTabItem(p.Name(), p.Content(win)),
)
// TUI:
_ = p.Model()
```

### Vlastní backend (Services)

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

### Dynamické načítání pluginů

```bash
make build-plugins              # build/plugins/*.so
PLUGIN_DIR=./build/plugins ./build/sysman
```

Vlastní `.so` plugin:

```go
// myplugin/main.go
package main

import "codeberg.org/oSoWoSo/SysMan/api"

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
| `make build` | všechny 14 binárky |
| `make build-sysman` | `build/sysman` — kompletní system manager |
| `make build-sysman-tui` | `build/sysman-tui` — kompletní system manager (TUI vstup) |
| `make build-serman` | `build/serman` — Services samostatně |
| `make build-serman-tui` | `build/serman-tui` — Services TUI (bez CGO) |
| `make build-ugsman` | `build/ugsman` — Users & Groups samostatně |
| `make build-ugsman-tui` | `build/ugsman-tui` — Users & Groups TUI (bez CGO) |
| `make build-infman` | `build/infman` — SysInfo samostatně |
| `make build-infman-tui` | `build/infman-tui` — SysInfo TUI (bez CGO) |
| `make build-srcman` | `build/srcman` — Templates samostatně |
| `make build-srcman-tui` | `build/srcman-tui` — Templates TUI (bez CGO) |
| `make build-pkgman` | `build/pkgman` — Packages samostatně |
| `make build-pkgman-tui` | `build/pkgman-tui` — Packages TUI (bez CGO) |
| `make build-vmsman` | `build/vmsman` — VM manager samostatně |
| `make build-vmsman-tui` | `build/vmsman-tui` — VM manager TUI (bez CGO) |
| `make build-plugins` | `build/plugins/*.so` — dynamické pluginy |
| `make test` | testy s detektorem závodů |
| `make lint` | `go vet` + golangci-lint |
| `make fmt` | `gofmt -s` |
| `make install` | instalace všech binárky + lang souborů |
| `make uninstall` | odstranění nainstalovaných souborů |
| `make release` | binárky + sha256 + tar.gz |
| `make clean` | smaže `build/` |

---

## Bezpečnost

Operace se službami (`sv`, aktivace/deaktivace symlinků) vyžadují vyšší oprávnění a jsou spouštěny přes `pkexec`, `doas`, nebo `sudo` (podle dostupnosti).

Volitelná konfigurace bez hesla (přidejte přes `visudo` nebo `/etc/doas.conf`):

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
