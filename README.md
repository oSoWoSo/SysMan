# svman – Service Manager for runit

**svman** is a simple service manager for systems using the **runit** init system. It offers a graphical (GUI) and text (TUI) interface for convenient management of symlinks in `/var/service`.

---

## Installation

### Prerequisites
- Go 1.18+
- runit installed and configured
- `sudo` for running commands with elevated privileges

### Compilation

```bash
git clone <repository>
cd svman
go build -o svman
```

### System installation

```bash
sudo cp svman /usr/local/bin/
sudo mkdir -p /usr/local/share/svman/lang
sudo cp lang/*.yaml /usr/local/share/svman/lang/
```

---

## Usage

### GUI mode (default)

```bash
svman
# or explicitly:
svman --gui
# or shortened:
svman -g
```

### TUI mode (terminal)

```bash
svman --tui
# or shortened:
svman -t
```

### Help

```bash
svman --help
# or shortened:
svman -h
```

---

## Controls

### GUI mode
- **Click on a service** – turns the service on/off
- **Right click** – displays details
- **Refresh** – reloads the list of services

### TUI mode

| Key | Action |
|---------|------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` / `Space` | Turn service on/off |
| `/` | Start search |
| `Tab` | Toggle filters (All / On / Off) |
| `r` | Reload service list |
| `Esc` / `q` / `Ctrl+C` | Exit application |

---

## Environment variables

| Variable | Description | Default |
|----------|-------|---------|
| `SERVICEDIR` | Path to service resources | `/etc/sv` |
| `SERVICEDESTDIR` | Path to enabled services | `/var/service` |
| `SVMAN_LANG` | Interface language (cs, en) | Detects from `LANG` |
| `LANGUAGE`, `LANG`, `LC_ALL`, `LC_MESSAGES` | Standard localization | – |

### Examples

```bash
# Custom service directory
SERVICEDIR=/home/user/services svman

# Czech language settings
SVMAN_LANG=cs svman

# Combination
SERVICEDIR=/opt/sv SVMAN_LANG=en svman --tui
```

---

## Architecture

### Modules

- **main.go** – Entry point, argument parsing
- **i18n.go** – Internationalization (Czech, English)
- **services.go** – Loading and managing services
- **gui.go** – Graphical interface (Fyne)
- **tui.go** – Text interface (Bubble Tea)
- **lang/*.yaml** – Language files

### Data flow

```
1. initI18n()        – Loads language files and detects language
2. loadServices()    – Scans SERVICEDIR and checks symlinks
3. GUI/TUI          – Displays list and waits for interaction
4. enableService()   – Creates symlink (sudo ln -s)
5. disableService()  – Deletes symlink (sudo rm)
6. Reload           – Reloads list
```

---

## Features

### ✅ Implemented
- Loading services from the runit directory
- Service status detection (on/off)
- Enabling/disabling services via sudo
- Filtering (All / On / Off)
- Real-time search
- Two language interfaces (Czech, English)
- Adaptive colors (light/dark mode)

### 🔄 Future extensions
- Direct administration without sudo (for privileged users)
- Displaying service logs
- Restarting/stopping/starting individual services
- Configuration via GUI
- Additional languages

---

## Security

**svman** uses `sudo` for operations requiring elevated privileges:
- Creating symlinks in `/var/service`
- Deleting symlinks

### Recommended sudo configuration

Add to `/etc/sudoers` (using `visudo`):

```sudoers
%wheel ALL=(ALL) NOPASSWD: /usr/bin/ln -s /etc/sv/* /var/service/*
%wheel ALL=(ALL) NOPASSWD: /usr/bin/rm /var/service/*
```

---

## Troubleshooting

### Application launches without GUI
Check that you have Fyne support installed:
```bash
go get fyne.io/fyne/v2
```

### Error: "Permission denied"
Check the sudo configuration and directory permissions.

### Services are not loading
Verify that `SERVICEDIR` points to the correct directory:
```bash
ls /etc/sv
```

### Language has not changed
Set `SVMAN_LANG` explicitly:
```bash
SVMAN_LANG=en svman
```

---

## Development

### Project structure

```
svman/
├── main.go           # Entry point
├── i18n.go           # Translations
├── services.go       # Service management
├── gui.go            # Fyne GUI
├── tui.go            # Bubble Tea TUI
├── lang/
│   ├── cs.yaml       # Czech translations
│   └── en.yaml       # English translations
├── go.mod
├── go.sum
└── README.md
```

### Dependencies

```go
github.com/charmbracelet/bubbletea    // TUI framework
github.com/charmbracelet/lipgloss      // Terminal styling
github.com/charmbracelet/bubbles       // TUI components
fyne.io/fyne/v2                        // GUI framework
gopkg.in/yaml.v3                       // YAML parsing
```

### Installing dependencies

```bash
go mod download
go mod tidy
```

---

## License

MIT License – see LICENSE file

---

## Contributions

Pull requests are welcome. For major changes, open an issue and discuss the changes in advance.

---

## Contact

Have a question or suggestion? Open a Codeberg issue.
