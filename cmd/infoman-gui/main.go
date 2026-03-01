// Command infoman-gui runs the System Info plugin as a standalone Fyne GUI.
package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"codeberg.org/oSoWoSo/SysMan/sysinfo"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(sysinfo.Usage)
			os.Exit(0)
		}
	}
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
