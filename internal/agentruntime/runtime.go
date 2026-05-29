package agentruntime

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
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
	UptimeSeconds uint64  `json:"uptimeSeconds"`
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryUsed    uint64  `json:"memoryUsed"`
	MemoryTotal   uint64  `json:"memoryTotal"`
	SwapUsed      uint64  `json:"swapUsed"`
	SwapTotal     uint64  `json:"swapTotal"`
	NetRxBytes    uint64  `json:"netRxBytes"`
	NetTxBytes    uint64  `json:"netTxBytes"`
	CollectedAt   string  `json:"collectedAt"`
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
	rx, tx := readNetDev()
	return RuntimeMetrics{
		UptimeSeconds: readUptime(),
		CPUPercent:    0,
		MemoryUsed:    subtract(memTotal, memAvailable),
		MemoryTotal:   memTotal,
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
