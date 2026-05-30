//go:build linux

package agentruntime

import "syscall"

func readDiskUsage(path string) (used uint64, total uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	total = stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	return subtract(total, free), total
}
