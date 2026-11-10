package common

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// TerminalLogo renders the shared SysMan logo as truecolor half-block cells.
// maxWidth is measured in terminal columns; an empty string means no logo file
// was found or it could not be decoded.
func TerminalLogo(maxWidth int) string {
	if maxWidth < 1 {
		return ""
	}
	for _, path := range logoPaths() {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		img, err := png.Decode(file)
		_ = file.Close()
		if err != nil {
			continue
		}
		return renderTerminalLogo(img, maxWidth)
	}
	return ""
}

func logoPaths() []string {
	var paths []string
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "void-transparent.png"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".config/fastfetch/void-transparent.png"),
			filepath.Join(home, ".dotfiles/config/fastfetch/void-transparent.png"),
		)
	}
	return append(paths, "/usr/share/pixmaps/void-transparent.png", "/usr/share/pixmaps/void-logo.png")
}

func renderTerminalLogo(img image.Image, maxWidth int) string {
	bounds := img.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 {
		return ""
	}
	width := min(maxWidth, bounds.Dx())
	pixelHeight := max(1, bounds.Dy()*width/bounds.Dx())
	cellHeight := (pixelHeight + 1) / 2
	var out strings.Builder
	for row := range cellHeight {
		for col := range width {
			x := bounds.Min.X + col*bounds.Dx()/width
			yTop := bounds.Min.Y + min(row*2, pixelHeight-1)*bounds.Dy()/pixelHeight
			yBottom := bounds.Min.Y + min(row*2+1, pixelHeight-1)*bounds.Dy()/pixelHeight
			top, topOK := opaqueColor(img.At(x, yTop))
			bottom, bottomOK := opaqueColor(img.At(x, yBottom))
			switch {
			case topOK && bottomOK:
				fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▀\x1b[0m", top.R, top.G, top.B, bottom.R, bottom.G, bottom.B)
			case topOK:
				fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%dm▀\x1b[0m", top.R, top.G, top.B)
			case bottomOK:
				fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%dm▄\x1b[0m", bottom.R, bottom.G, bottom.B)
			default:
				out.WriteByte(' ')
			}
		}
		if row+1 < cellHeight {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func opaqueColor(c color.Color) (color.NRGBA, bool) {
	v := color.NRGBAModel.Convert(c).(color.NRGBA)
	return v, v.A >= 32
}
