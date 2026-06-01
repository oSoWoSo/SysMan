//go:build !tui_only

package common

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

// NewApp creates a Fyne application with the shared SysMan app ID and
// declares migration to the fyne.Do threading model. All UI updates from
// goroutines must be wrapped in fyne.Do — see https://docs.fyne.io/started/goroutines
func NewApp(name string) fyne.App {
	app.SetMetadata(fyne.AppMetadata{
		ID:      AppID,
		Name:    name,
		Version: Version,
		Migrations: map[string]bool{
			"fyneDo": true,
		},
	})
	return app.NewWithID(AppID)
}
