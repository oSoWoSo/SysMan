package plugin

import (
	"fmt"
	"os/exec"
	"sort"
)

// XbpsSrc represents a package in the xbps-src build system.
type XbpsSrc struct {
	Name string
}

// LoadXbpsSrcPackages lists available packages from the xbps-src repository.
func LoadXbpsSrcPackages(repoDir string) []XbpsSrc {
	// Placeholder: real implementation would parse pkgbuild files.
	return []XbpsSrc{{Name: "example"}}
}

// BuildPackage runs xbps-src build for a given package name.
func BuildPackage(pkg string, repoDir string) error {
	cmd := exec.Command("xbps-src", "pkgcreate", pkg)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %s", string(out))
	}
	return nil
}

// String returns the package name.
func (p XbpsSrc) String() string { return p.Name }

// SortXbpsSrc sorts packages alphabetically by name.
func SortXbpsSrc(pkgs []XbpsSrc) {
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })
}
