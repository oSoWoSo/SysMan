package common

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/storage"
)

// AppIcon tries to load the application icon from well-known paths.
// Returns nil when no icon is found.
func AppIcon() *fyne.StaticResource {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, "void-transparent.png"))
	}
	candidates = append(candidates,
		"/usr/local/share/SysMan/void-transparent.png",
		"/usr/share/SysMan/void-transparent.png",
	)
	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil {
			return &fyne.StaticResource{
				StaticName:    "void-transparent.png",
				StaticContent: data,
			}
		}
	}
	return nil
}

// SetWindowIcon loads and sets the window icon from well-known paths.
// Returns true if an icon was found and set.
func SetWindowIcon(win fyne.Window) bool {
	if icon := AppIcon(); icon != nil {
		win.SetIcon(icon)
		return true
	}
	return false
}

// LogoImage tries to load a distro logo PNG from well-known paths for display in About dialogs.
// Returns nil when no logo is found.
func LogoImage() *canvas.Image {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, "void-transparent.png"))
	}
	candidates = append(candidates,
		os.ExpandEnv("$HOME/.config/fastfetch/void-transparent.png"),
		os.ExpandEnv("$HOME/.dotfiles/config/fastfetch/void-transparent.png"),
		"/usr/share/pixmaps/void-transparent.png",
		"/usr/share/pixmaps/void-logo.png",
	)
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			img := canvas.NewImageFromURI(storage.NewFileURI(path))
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(200, 200))
			return img
		}
	}
	return nil
}
