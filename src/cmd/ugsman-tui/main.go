// Command ugman-tui is a standalone TUI for user and group management.
package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/SysMan/src/ugsman"
)

func main() {
	ugsman.InitI18n()
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(ugsman.Usage)
			os.Exit(0)
		}
	}
	ugsman.RunTUI()
}
