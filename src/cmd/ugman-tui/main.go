// Command ugman-tui is a standalone TUI for user and group management.
package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/SysMan/src/usergroups"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(usergroups.Usage)
			os.Exit(0)
		}
	}
	usergroups.RunTUI()
}
