package common

import (
	"fmt"
	"os/exec"
	"strings"
)

// CopyToClipboard copies the given text to the system clipboard.
// Tries xclip, xsel, and wl-copy in order. Returns error if none available.
func CopyToClipboard(text string) error {
	for _, tool := range []struct {
		cmd  string
		args []string
	}{
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
		{"wl-copy", nil},
	} {
		cmd := exec.Command(tool.cmd, tool.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)")
}
