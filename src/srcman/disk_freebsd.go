//go:build freebsd

package srcman

import "syscall"

// diskFreeTotal returns free and total bytes for the filesystem containing dir.
func diskFreeTotal(dir string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return 0, 0, err
	}
	// FreeBSD Statfs_t: Bavail is int64, Bsize/Blocks are uint64.
	avail := uint64(stat.Bavail) * stat.Bsize
	total := stat.Blocks * stat.Bsize
	return avail, total, nil
}
