// Package main is the svman plugin entry point for dynamic loading.
//
// Build as a shared library:
//
//	go build -buildmode=plugin -o plugins/svman.so ./pluginentry/svman/
//
// The system manager loads this .so and calls New() via plugin.Lookup("New").
// Service directories are read from SERVICEDIR / SERVICEDESTDIR env vars.
package main

import (
	"os"

	"codeberg.org/oSoWoSo/svman/api"
	svman "codeberg.org/oSoWoSo/svman/plugin"
)

// New is the plugin factory called by the system manager after loading the .so.
// It satisfies the func() api.PluginIF signature expected by loadDynamic.
func New() api.PluginIF {
	serviceDir := os.Getenv("SERVICEDIR")
	if serviceDir == "" {
		serviceDir = svman.DefaultServiceDir
	}
	serviceDestDir := os.Getenv("SERVICEDESTDIR")
	if serviceDestDir == "" {
		serviceDestDir = svman.DefaultServiceDestDir
	}
	svman.InitI18n()
	return svman.New(serviceDir, serviceDestDir)
}
