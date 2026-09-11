//go:build openbsd

package srcman

import "syscall"

// diskFreeTotal returns free and total bytes for the filesystem containing dir.
func diskFreeTotal(dir string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return 0, 0, err
	}
	// OpenBSD Statfs_t uses F_* prefixed fields; F_bavail is int64,
	// F_blocks is uint64 and F_bsize is uint32.
	avail := uint64(stat.F_bavail) * uint64(stat.F_bsize)
	total := stat.F_blocks * uint64(stat.F_bsize)
	return avail, total, nil
}
