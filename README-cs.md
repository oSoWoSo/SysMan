# svman – Service Manager for runit

**svman** je jednoduchý správce služeb pro systémy používající **runit** init systém. Nabízí grafické (GUI) a textové (TUI) rozhraní pro pohodlnou správu symlinků v `/var/service`.

---

## Instalace

### Předpoklady
- Go 1.18+
- runit nainstalovaný a nakonfigurovaný
- `sudo` pro spouštění příkazů se zvýšenými právy

### Kompilace

```bash
git clone <repository>
cd svman
go build -o svman
```

### Systémová instalace

```bash
sudo cp svman /usr/local/bin/
sudo mkdir -p /usr/local/share/svman/lang
sudo cp lang/*.yaml /usr/local/share/svman/lang/
```

---

## Použití

### GUI režim (výchozí)

```bash
svman
# nebo explicitně:
svman --gui
# nebo zkráceně:
svman -g
```

### TUI režim (terminál)

```bash
svman --tui
# nebo zkráceně:
svman -t
```

### Nápověda

```bash
svman --help
# nebo zkráceně:
svman -h
```

---

## Ovládání

### GUI režim
- **Kliknutí na službu** – zapne/vypne službu
- **Pravé tlačítko** – zobrazí detaily
- **Refresh** – znovu načte seznam služeb

### TUI režim

| Klávesa | Akce |
|---------|------|
| `↑` / `k` | Posun nahoru |
| `↓` / `j` | Posun dolů |
| `Enter` / `Space` | Zapne/vypne službu |
| `/` | Spustí vyhledávání |
| `Tab` | Přepíná filtry (Vše / Zapnuto / Vypnuto) |
| `r` | Znovu načte seznam služeb |
| `Esc` / `q` / `Ctrl+C` | Ukončí aplikaci |

---

## Proměnné prostředí

| Proměnná | Popis | Výchozí |
|----------|-------|---------|
| `SERVICEDIR` | Cesta ke zdrojům služeb | `/etc/sv` |
| `SERVICEDESTDIR` | Cesta k povolených služeb | `/var/service` |
| `SVMAN_LANG` | Jazyk rozhraní (cs, en) | Detekuje z `LANG` |
| `LANGUAGE`, `LANG`, `LC_ALL`, `LC_MESSAGES` | Standardní lokalizace | – |

### Příklady

```bash
# Vlastní adresář služeb
SERVICEDIR=/home/user/services svman

# Nastavení češtiny
SVMAN_LANG=cs svman

# Kombinace
SERVICEDIR=/opt/sv SVMAN_LANG=en svman --tui
```

---

## Architektura

### Moduly

- **main.go** – Entry point, parsování argumentů
- **i18n.go** – Internacionalizace (čeština, angličtina)
- **services.go** – Načítání a správa služeb
- **gui.go** – Grafické rozhraní (Fyne)
- **tui.go** – Textové rozhraní (Bubble Tea)
- **lang/*.yaml** – Jazykové soubory

### Tok dat

```
1. initI18n()        – Načte jazykové soubory a detekuje jazyk
2. loadServices()    – Skenuje SERVICEDIR a kontroluje symlinky
3. GUI/TUI          – Zobrazí seznam a čeká na interakci
4. enableService()   – Vytvoří symlink (sudo ln -s)
5. disableService()  – Smaže symlink (sudo rm)
6. Reload           – Znovu načte seznam
```

---

## Funkce

### ✅ Implementováno
- Načítání služeb z runit adresáře
- Detekce stavu služby (zapnuto/vypnuto)
- Zapnutí/vypnutí služby přes sudo
- Filtrování (Vše / Zapnuto / Vypnuto)
- Vyhledávání v reálném čase
- Dvě jazyková rozhraní (čeština, angličtina)
- Adaptivní barvy (light/dark režim)

### 🔄 Budoucí rozšíření
- Přímá správa bez sudo (pro privilegované uživatele)
- Zobrazení logů služby
- Restart/stop/start jednotlivých služeb
- Konfigurace skrz GUI
- Další jazyky

---

## Bezpečnost

**svman** používá `sudo` pro operace vyžadující zvýšená práva:
- Vytváření symlinků v `/var/service`
- Mazání symlinků

### Doporučená konfigurace sudo

Přidejte do `/etc/sudoers` (pomocí `visudo`):

```sudoers
%wheel ALL=(ALL) NOPASSWD: /usr/bin/ln -s /etc/sv/* /var/service/*
%wheel ALL=(ALL) NOPASSWD: /usr/bin/rm /var/service/*
```

---

## Řešení problémů

### Aplikace se spouští bez GUI
Zkontrolujte, zda máte nainstalovanou Fyne podporu:
```bash
go get fyne.io/fyne/v2
```

### Chyba: "Permission denied"
Zkontrolujte sudo konfiguraci a práva na adresářích.

### Služby se nenačítají
Ověřte, že `SERVICEDIR` ukazuje na správný adresář:
```bash
ls /etc/sv
```

### Jazyk se nezměnil
Nastavte explicitně `SVMAN_LANG`:
```bash
SVMAN_LANG=cs svman
```

---

## Vývoj

### Struktura projektu

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

### Závislosti

```go
github.com/charmbracelet/bubbletea    // TUI framework
github.com/charmbracelet/lipgloss      // Terminal styling
github.com/charmbracelet/bubbles       // TUI components
fyne.io/fyne/v2                        // GUI framework
gopkg.in/yaml.v3                       // YAML parsing
```

### Instalace závislostí

```bash
go mod download
go mod tidy
```

---

## Licence

MIT License – viz LICENSE soubor

---

## Příspěvky

Pull requesty jsou vítány. Pro větší změny otevřete issue a diskutujte změny předem.

---

## Kontakt

Máte otázku nebo návrh? Otevřete Codeberg issue.
