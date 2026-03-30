// Package main is the xbps package manager plugin entry point for dynamic loading.
//
// Build as a shared library:
//
//	go build -buildmode=plugin -o plugins/xbps-pkg.so ./pluginentry/xbps-pkg/
//
// The system manager loads this .so and calls New() via plugin.Lookup("New").
package main

import (
	"codeberg.org/oSoWoSo/SysMan/api"
	xbpspkg "codeberg.org/oSoWoSo/SysMan/xbps-pkg"
)

// New is the plugin factory called by the system manager after loading the .so.
func New() api.PluginIF {
	return xbpspkg.NewXbps()
}
