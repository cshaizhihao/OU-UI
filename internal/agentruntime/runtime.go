package agentruntime

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SystemInfo struct {
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Kernel      string `json:"kernel"`
	CPUModel    string `json:"cpuModel"`
	CPUCount    int    `json:"cpuCount"`
	MemoryTotal uint64 `json:"memoryTotal"`
	SwapTotal   uint64 `json:"swapTotal"`
}

type RuntimeMetrics struct {
	UptimeSeconds uint64              `json:"uptimeSeconds"`
	CPUPercent    float64             `json:"cpuPercent"`
	MemoryUsed    uint64              `json:"memoryUsed"`
	MemoryTotal   uint64              `json:"memoryTotal"`
	DiskUsed      uint64              `json:"diskUsed"`
	DiskTotal     uint64              `json:"diskTotal"`
	SwapUsed      uint64              `json:"swapUsed"`
	SwapTotal     uint64              `json:"swapTotal"`
	NetRxBytes    uint64              `json:"netRxBytes"`
	NetTxBytes    uint64              `json:"netTxBytes"`
	NetRxRateBps  uint64              `json:"netRxRateBps"`
	NetTxRateBps  uint64              `json:"netTxRateBps"`
	CollectedAt   string              `json:"collectedAt"`
	NodeTraffic   []NodeTrafficMetric `json:"nodeTraffic"`
}

type NodeTrafficMetric struct {
	NodeID      string `json:"nodeId"`
	Name        string `json:"name"`
	RxBytes     uint64 `json:"rxBytes"`
	TxBytes     uint64 `json:"txBytes"`
	RxRateBps   uint64 `json:"rxRateBps"`
	TxRateBps   uint64 `json:"txRateBps"`
	Connections int    `json:"connections"`
	CollectedAt string `json:"collectedAt"`
}

type Sampler struct {
	mu               sync.Mutex
	dataDir          string
	lastCPU          cpuTimes
	hasCPU           bool
	lastRx           uint64
	lastTx           uint64
	lastAt           time.Time
	lastNodeCounters map[string]nodeCounter
	lastManagedNodes []ManagedNodeRef
}

func NewSampler() *Sampler {
	return NewSamplerWithDataDir("")
}

func NewSamplerWithDataDir(dataDir string) *Sampler {
	return &Sampler{dataDir: dataDir, lastNodeCounters: map[string]nodeCounter{}}
}

func (s *Sampler) Collect() RuntimeMetrics {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics := CollectRuntimeMetrics()
	now := time.Now()
	rx, tx := metrics.NetRxBytes, metrics.NetTxBytes

	if currentCPU, ok := readCPUTimes(); ok {
		if s.hasCPU && currentCPU.total >= s.lastCPU.total && currentCPU.idle >= s.lastCPU.idle {
			totalDelta := currentCPU.total - s.lastCPU.total
			idleDelta := currentCPU.idle - s.lastCPU.idle
			if totalDelta > 0 && idleDelta <= totalDelta {
				metrics.CPUPercent = float64(totalDelta-idleDelta) * 100 / float64(totalDelta)
			}
		}
		s.lastCPU = currentCPU
		s.hasCPU = true
	}

	if !s.lastAt.IsZero() {
		elapsed := now.Sub(s.lastAt).Seconds()
		if elapsed > 0 {
			metrics.NetRxRateBps = uint64(float64(subtract(rx, s.lastRx)) / elapsed)
			metrics.NetTxRateBps = uint64(float64(subtract(tx, s.lastTx)) / elapsed)
		}
	}
	s.lastRx = rx
	s.lastTx = tx
	s.lastAt = now
	metrics.NodeTraffic = s.collectNodeTraffic(now)
	return metrics
}

func CollectSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()
	memTotal, _, swapTotal, _ := readMemInfo()
	return SystemInfo{
		Hostname:    hostname,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Kernel:      readFirstLine("/proc/version"),
		CPUModel:    readCPUModel(),
		CPUCount:    runtime.NumCPU(),
		MemoryTotal: memTotal,
		SwapTotal:   swapTotal,
	}
}

func CollectRuntimeMetrics() RuntimeMetrics {
	memTotal, memAvailable, swapTotal, swapFree := readMemInfo()
	diskUsed, diskTotal := readDiskUsage("/")
	rx, tx := readNetDev()
	return RuntimeMetrics{
		UptimeSeconds: readUptime(),
		CPUPercent:    0,
		MemoryUsed:    subtract(memTotal, memAvailable),
		MemoryTotal:   memTotal,
		DiskUsed:      diskUsed,
		DiskTotal:     diskTotal,
		SwapUsed:      subtract(swapTotal, swapFree),
		SwapTotal:     swapTotal,
		NetRxBytes:    rx,
		NetTxBytes:    tx,
		CollectedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

func readMemInfo() (memTotal, memAvailable, swapTotal, swapFree uint64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		value *= 1024
		switch strings.TrimSuffix(fields[0], ":") {
		case "MemTotal":
			memTotal = value
		case "MemAvailable":
			memAvailable = value
		case "SwapTotal":
			swapTotal = value
		case "SwapFree":
			swapFree = value
		}
	}
	return memTotal, memAvailable, swapTotal, swapFree
}

func readNetDev() (uint64, uint64) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var rxTotal, txTotal uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		name := strings.TrimSpace(parts[0])
		if name == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		rxTotal += rx
		txTotal += tx
	}
	return rxTotal, txTotal
}

func readUptime() uint64 {
	content, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(content))
	if len(fields) == 0 {
		return 0
	}
	value, _ := strconv.ParseFloat(fields[0], 64)
	return uint64(value)
}

func readCPUModel() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return runtime.GOARCH
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Hardware") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return runtime.GOARCH
}

type cpuTimes struct {
	idle  uint64
	total uint64
}

func readCPUTimes() (cpuTimes, bool) {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTimes{}, false
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return cpuTimes{}, false
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimes{}, false
	}
	var values []uint64
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuTimes{}, false
		}
		values = append(values, value)
	}
	var total uint64
	for _, value := range values {
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return cpuTimes{idle: idle, total: total}, true
}

func readFirstLine(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(content))
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		return line[:idx]
	}
	return line
}

func subtract(a, b uint64) uint64 {
	if b > a {
		return 0
	}
	return a - b
}
