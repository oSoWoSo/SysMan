// Command infoman-gui runs the System Info plugin as a standalone Fyne GUI.
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"codeberg.org/oSoWoSo/SysMan/sysinfo"
)

func main() {
	p := sysinfo.New()
	a := app.New()
	win := a.NewWindow(p.Name())
	win.SetContent(p.Content(win))
	win.Resize(fyne.NewSize(420, 300))
	win.SetMaster()
	win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			a.Quit()
		}
	})
	win.ShowAndRun()
}
