//go:build linux

package srcman

import "syscall"

// diskFreeTotal returns free and total bytes for the filesystem containing dir.
func diskFreeTotal(dir string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return 0, 0, err
	}
	avail := stat.Bavail * uint64(stat.Bsize) //nolint:unconvert
	total := stat.Blocks * uint64(stat.Bsize) //nolint:unconvert
	return avail, total, nil
}
