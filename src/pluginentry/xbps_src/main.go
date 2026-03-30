// Plugin entry point for dynamic loading of the xbps-src plugin.
// Build with: go build -buildmode=plugin -o build/plugins/xbps.so ./pluginentry/xbps-src/
package main

import (
	"codeberg.org/oSoWoSo/SysMan/src/api"
	xbpssrc "codeberg.org/oSoWoSo/SysMan/src/xbps_src"
)

// New returns a new xbps-src plugin instance.
// Called by the system manager via plugin.Lookup("New").
func New() api.PluginIF {
	return xbpssrc.New(xbpssrc.DefaultDistDir)
}
