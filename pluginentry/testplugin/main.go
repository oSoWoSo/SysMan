// Package main is the testplugin entry point for dynamic loading.
//
// Build as a shared library:
//
//	go build -buildmode=plugin -o plugins/testplugin.so ./pluginentry/testplugin/
//
// The system manager loads this .so and calls New() via plugin.Lookup("New").
package main

import (
	"codeberg.org/oSoWoSo/svman/api"
	"codeberg.org/oSoWoSo/svman/testplugin"
)

// New is the plugin factory called by the system manager after loading the .so.
// It satisfies the func() api.PluginIF signature expected by loadDynamic.
func New() api.PluginIF {
	return testplugin.New()
}
