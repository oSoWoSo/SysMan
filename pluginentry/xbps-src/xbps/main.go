// Plugin entry point for dynamic loading of the xbps-src plugin.
// Build with: go build -buildmode=plugin -o build/plugins/xbps.so ./pluginentry/xbps/
package main

import (
	"codeberg.org/oSoWoSo/svman/api"
	"codeberg.org/oSoWoSo/svman/xbpsplugin"
)

// New returns a new xbps-src plugin instance.
// Called by the system manager via plugin.Lookup("New").
func New() api.PluginIF {
	return xbpsplugin.New(xbpsplugin.DefaultDistDir)
}
