// Package main is the sysinfo plugin entry point for dynamic loading.
//
// Build as a shared library:
//
//	go build -buildmode=plugin -o plugins/sysinfo.so ./pluginentry/sysinfo/
//
// The system manager loads this .so and calls New() via plugin.Lookup("New").
package main

import (
	"codeberg.org/oSoWoSo/SysMan/api"
	"codeberg.org/oSoWoSo/SysMan/sysinfo"
)

// New is the plugin factory called by the system manager after loading the .so.
// It satisfies the func() api.PluginIF signature expected by loadDynamic.
func New() api.PluginIF {
	return sysinfo.New()
}
