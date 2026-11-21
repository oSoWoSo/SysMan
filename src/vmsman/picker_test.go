package vmman

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestPickerOnlyShowsRealChoicesPorted(t *testing.T) {
	native := test.NewApp()
	defer native.Quit()
	selector := widget.NewSelect(nil, nil)
	row := container.NewVBox(selector)
	SetPickerOptions(selector, row, []string{"standard"})
	if row.Visible() || selector.Selected != "standard" {
		t.Fatal("fixed edition should be hidden but retained")
	}
	SetPickerOptions(selector, row, []string{"standard", "extended"})
	if !row.Visible() || selector.Selected != "standard" {
		t.Fatal("multiple editions should be selectable")
	}
	selector.SetSelected("extended")
	SetPickerOptions(selector, row, nil)
	if row.Visible() || selector.Selected != "" {
		t.Fatal("absent editions should clear any previous selection")
	}
}

func TestPickerSkipsFixedCombinationsPorted(t *testing.T) {
	for _, editions := range [][]string{nil, {"standard"}} {
		if PickerNeedsInput([]string{"amd64"}, []string{"latest"}, editions) {
			t.Fatal("fixed combination should download without opening a picker")
		}
	}
	for _, choices := range [][3][]string{
		{{"amd64", "arm64"}, {"latest"}, nil},
		{{"amd64"}, {"old", "latest"}, nil},
		{{"amd64"}, {"latest"}, {"standard", "extended"}},
	} {
		if !PickerNeedsInput(choices[0], choices[1], choices[2]) {
			t.Fatal("real choices should open a picker")
		}
	}
}
