package vmman

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// iconTile is a native tappable tile for supported/ready VMs.
type iconTile struct {
	widget.BaseWidget
	name              string
	icon              fyne.Resource
	showText, hovered bool
	onTapped, onHover func()
}

func newIconTile(name string, icon fyne.Resource, showText bool, tapped func()) *iconTile {
	t := &iconTile{name: name, icon: icon, showText: showText, onTapped: tapped}
	t.ExtendBaseWidget(t)
	return t
}

func (t *iconTile) Tapped(*fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

func (t *iconTile) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (t *iconTile) MouseIn(*desktop.MouseEvent) {
	t.hovered = true
	t.Refresh()
	if t.onHover != nil {
		t.onHover()
	}
}

func (t *iconTile) MouseMoved(*desktop.MouseEvent) {}

func (t *iconTile) MouseOut() {
	t.hovered = false
	t.Refresh()
}

func (t *iconTile) CreateRenderer() fyne.WidgetRenderer {
	image := canvas.NewImageFromResource(t.icon)
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.NewSize(52, 52))
	content := container.NewCenter(image)
	if t.showText {
		label := widget.NewLabel(t.name)
		label.Alignment = fyne.TextAlignCenter
		label.Wrapping = fyne.TextWrapWord
		content = container.NewBorder(nil, label, nil, nil, content)
	}
	return widget.NewSimpleRenderer(container.NewPadded(content))
}
