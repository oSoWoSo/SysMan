package common

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

// AboutConfig holds the parameters for the About dialog.
type AboutConfig struct {
	Win       fyne.Window
	Title     string // app title (e.g. t("app.title"))
	Subtitle  string // app subtitle (e.g. t("app.subtitle"))
	Version   string // version string
	Author    string // author string
	License   string // license string
	URL       string // repository/documentation URL
	URLLabel  string // label for the hyperlink (empty = URL itself)
	DialogBtn string // dialog button label (e.g. t("btn.about"))
	CloseBtn  string // close button label (e.g. t("btn.close"))
}

// ShowAbout displays a standard About dialog using the given configuration.
func ShowAbout(cfg AboutConfig) {
	title := canvas.NewText(cfg.Title, color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	title.TextSize = 26
	title.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

	subtitle := canvas.NewText(cfg.Subtitle, color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff})
	subtitle.TextSize = 12

	infoForm := widget.NewForm(
		widget.NewFormItem("Version", widget.NewLabel(cfg.Version)),
		widget.NewFormItem("Author", widget.NewLabel(cfg.Author)),
		widget.NewFormItem("License", widget.NewLabel(cfg.License)),
	)

	repoURL, _ := url.Parse(cfg.URL)
	linkLabel := cfg.URLLabel
	if linkLabel == "" {
		linkLabel = cfg.URL
	}
	link := widget.NewHyperlink(linkLabel, repoURL)

	content := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		infoForm,
		container.NewCenter(link),
	)

	d := dialog.NewCustom(cfg.DialogBtn, cfg.CloseBtn, content, cfg.Win)
	d.Show()
}
