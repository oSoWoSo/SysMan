# SysMan GUI Unification Plan

## Nová struktura

```
src/common/              (package common — sloučení tui/ + config/)
├── about.go             # showAbout helper
├── filter.go            # generic Filter[T] + FilterMode
├── hover.go             # HoverableButton
├── icon.go              # AppIcon helper
├── ansi.go              # ANSI colors
├── i18n.go              # i18n system
├── config.go            # configuration
└── version.go           # Version + AppAuthor + AppLicense + AppURL
```

> **POZOR**: Všechny soubory v `src/common/` MUSÍ mít `package common`.
> Aktuální stav je nefunkční — `config.go`/`i18n.go` mají `package config`,
> `hover_button.go`/`ansi.go` mají `package tui`. Commit #1 to opraví.

---

## Zjištěné problémy

- `ansi.go` importuje `infman.AnsiRe` — při přesunu do common hrozí cyklická závislost.
  Řešení: přesunout regex definici do `common/ansi.go`.
- `Filter[T]` je duplicitní ve 3 modulech (pkgman, serman, vmsman) — všechny mají stejnou signaturu.
- `showAbout()` existuje v 6 různých implementacích se stejnou strukturou.
- Package name `vmman` (bez 's') je záměrný — neměnit.
- `infman` nemá `RunGUI()`, jeho GUI se dodává přes `Plugin.Content()`.

---

## Seznam commitů

| # | Commit | Popis | Stav |
|---|--------|-------|------|
| 1 | refactor: unify common package name | Všechny soubory v src/common/ → `package common`, oprava importů | IN PROGRESS |
| 2 | refactor: move generic Filter to common | common/filter.go (z pkgman/serman/vmsman), aktualizace importů | Pending |
| 3 | refactor: move version info to common | common/version.go (Version, AppAuthor, AppLicense, AppURL), ldflags → common.Version | Pending |
| 4 | refactor: add common showAbout helper | common/about.go, aktualizace všech 6 modulů | Pending |
| 5 | add: application icon helper | common/icon.go + SetIcon na všech GUI entry points | Pending |
| 6 | add: tooltips to vmsman language files | src/lang/vmsman/{en,cs}.yaml | Pending |
| 7 | refactor: vmsman uses common components | hover, tooltips, about, statusBar | Pending |
| 8 | refactor: serman uses common components | hover, about, version | Pending |
| 9 | refactor: ugsman uses common components | hover, about, version | Pending |
| 10 | add: statusBar and tooltips to infman | infman/sysinfo.go | Pending |
| 11 | add: show all module versions in sysman about | sysman-gui/main.go — About dialog s verzemi | Pending |
| 12 | chore: bump version to 0.008 Alpha | Makefile + common/version.go | Pending |

---

## Analýza současného stavu

### Moduly

| Modul | statusBar | btnAbout | tooltips | HoverableButton |
|-------|-----------|----------|----------|-----------------|
| ugsman | Y | Y | Y | vlastní impl |
| serman | Y | partial | partial | vlastní impl |
| pkgman | Y | Y | Y | tui.HoverableButton |
| srcman | Y | Y | Y | tui.HoverableButton |
| vmsman | Y | N | N | N |
| infman | N | partial | N | N/A |

### Verze

| Modul | Version | Zdroj |
|-------|---------|-------|
| serman | "0.001 Alpha" | vlastní var |
| vmsman | "0.001 Alpha" | vlastní var |
| pkgman | serman.Version | import |
| srcman | serman.Version | import |
| ugsman | serman.Version | import |
| infman | serman.Version | import |
| sysman | serman.Version | ldflags |

### Soubory importující tui/

| Soubor | Používá |
|--------|---------|
| src/pkgman/plugin_gui.go | HoverableButton, HasAnsiCodes, AnsiToRichSegments |
| src/srcman/gui.go | HoverableButton |
| src/srcman/output_widget.go | HasAnsiCodes, AnsiToRichSegments |

### Soubory importující config/

| Soubor | Používá |
|--------|---------|
| src/srcman/config.go | LoadSysManConfig, SaveSysManConfig |
| src/vmsman/plugin.go | LoadSysManConfig (alias commonconfig) |

---

## Jazykové soubory k aktualizaci

### vmsman/en.yaml — přidat:
```yaml
  btn.kill: Kill
  btn.connect: Connect
  tooltip.boot: Boot selected VM
  tooltip.kill: Stop/Kill the VM
  tooltip.connect: Connect to VM via SPICE
  tooltip.about: About this application
```

### vmsman/cs.yaml — přidat:
```yaml
  btn.kill: Zabít
  btn.connect: Připojit
  tooltip.boot: Spustit vybrané VM
  tooltip.kill: Zastavit/zabít VM
  tooltip.connect: Připojit k VM přes SPICE
  tooltip.about: O této aplikaci
```

---

## Soubory k úpravě

### GUI Entry Points (přidat SetIcon):
- src/cmd/infman-gui/main.go
- src/cmd/serman-gui/main.go
- src/cmd/sysman-gui/main.go
- src/cmd/ugsman-gui/main.go
- src/cmd/pkgman-gui/main.go
- src/cmd/srcman-gui/main.go
- src/vmsman/gui.go (RunGUI)
- src/ugsman/gui.go (RunGUI)
- src/serman/gui.go (RunGUI)
- src/pkgman/plugin_gui.go (RunGUI)
- src/srcman/gui.go (RunGUI)

### Moduly GUI:
- src/serman/gui.go — remove duplicate hoverableButton, add tooltips
- src/ugsman/gui.go — remove duplicate hoverableButton
- src/vmsman/gui.go — add tooltips, btnAbout, use common components
- src/pkgman/plugin_gui.go — update tui → common imports
- src/srcman/gui.go — update tui → common imports
- src/srcman/output_widget.go — update tui → common imports
- src/infman/sysinfo.go — add statusBar, tooltips

### Jazykové soubory:
- src/lang/vmsman/en.yaml
- src/lang/vmsman/cs.yaml

---

## Pravidla

- Pracovat na feature branchi `refactor/common-unify`
- Každý commit musí kompilovat (`go build ./...`)
- Po posledním commitu tag + merge do main
- Celkem: 12 commitů + 1 TAG
