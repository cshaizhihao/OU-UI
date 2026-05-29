package systeminfo

import (
	"os"
	"runtime"
	"time"
)

type Snapshot struct {
	Hostname         string    `json:"hostname"`
	OS               string    `json:"os"`
	Arch             string    `json:"arch"`
	CPUPercent       float64   `json:"cpu_percent"`
	MemoryUsedBytes  uint64    `json:"memory_used_bytes"`
	MemoryTotalBytes uint64    `json:"memory_total_bytes"`
	DiskUsedBytes    uint64    `json:"disk_used_bytes"`
	DiskTotalBytes   uint64    `json:"disk_total_bytes"`
	CollectedAt      time.Time `json:"collected_at"`
}

type Collector interface {
	Collect() (Snapshot, error)
}

type BasicCollector struct{}

func (BasicCollector) Collect() (Snapshot, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return Snapshot{
		Hostname:         hostname,
		OS:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		CPUPercent:       0,
		MemoryUsedBytes:  mem.Alloc,
		MemoryTotalBytes: mem.Sys,
		DiskUsedBytes:    0,
		DiskTotalBytes:   0,
		CollectedAt:      time.Now().UTC(),
	}, nil
}
