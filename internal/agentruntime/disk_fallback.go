//go:build !linux

package agentruntime

func readDiskUsage(string) (used uint64, total uint64) {
	return 0, 0
}
