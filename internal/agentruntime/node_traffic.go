package agentruntime

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const nodeTrafficQueryConcurrency = 4

var (
	readSystemdIPCounters    = systemdIPCounters
	countNodePortConnections = countPortConnections
)

type nodeCounter struct {
	rx uint64
	tx uint64
	at time.Time
}

func (s *Sampler) collectNodeTraffic(now time.Time) []NodeTrafficMetric {
	nodes, err := LoadManagedNodes(s.dataDir)
	if err != nil {
		nodes = s.lastManagedNodes
	} else {
		s.lastManagedNodes = nodes
	}
	if len(nodes) == 0 {
		return nil
	}
	if s.lastNodeCounters == nil {
		s.lastNodeCounters = map[string]nodeCounter{}
	}
	readings := readNodeTrafficCounters(nodes)
	metrics := make([]NodeTrafficMetric, 0, len(nodes))
	for _, reading := range readings {
		if !reading.ok {
			continue
		}
		node := reading.node
		last, hasLast := s.lastNodeCounters[node.NodeID]
		var rxRate, txRate uint64
		if hasLast && reading.rx >= last.rx && reading.tx >= last.tx {
			elapsed := now.Sub(last.at).Seconds()
			if elapsed > 0 {
				rxRate = uint64(float64(reading.rx-last.rx) / elapsed)
				txRate = uint64(float64(reading.tx-last.tx) / elapsed)
			}
		}
		s.lastNodeCounters[node.NodeID] = nodeCounter{rx: reading.rx, tx: reading.tx, at: now}
		metrics = append(metrics, NodeTrafficMetric{
			NodeID:      node.NodeID,
			Name:        node.Name,
			RxBytes:     reading.rx,
			TxBytes:     reading.tx,
			RxRateBps:   rxRate,
			TxRateBps:   txRate,
			Connections: countNodePortConnections(node.Port),
			CollectedAt: now.UTC().Format(time.RFC3339),
		})
	}
	return metrics
}

type nodeTrafficReading struct {
	index int
	node  ManagedNodeRef
	rx    uint64
	tx    uint64
	ok    bool
}

func readNodeTrafficCounters(nodes []ManagedNodeRef) []nodeTrafficReading {
	readings := make([]nodeTrafficReading, len(nodes))
	sem := make(chan struct{}, nodeTrafficQueryConcurrency)
	results := make(chan nodeTrafficReading, len(nodes))
	var wg sync.WaitGroup
	for i, node := range nodes {
		i, node := i, node
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rx, tx, ok := readSystemdIPCounters(node.ServiceName)
			results <- nodeTrafficReading{index: i, node: node, rx: rx, tx: tx, ok: ok}
		}()
	}
	wg.Wait()
	close(results)
	for reading := range results {
		readings[reading.index] = reading
	}
	return readings
}

func systemdIPCounters(serviceName string) (rx, tx uint64, ok bool) {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return 0, 0, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "systemctl", "show", serviceName, "--property=IPIngressBytes", "--property=IPEgressBytes").Output()
	if err != nil {
		return 0, 0, false
	}
	values := parseSystemdProperties(string(output))
	rx, rxOK := parseSystemdCounter(values["IPIngressBytes"])
	tx, txOK := parseSystemdCounter(values["IPEgressBytes"])
	return rx, tx, rxOK && txOK
}

func parseSystemdProperties(output string) map[string]string {
	values := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return values
}

func parseSystemdCounter(value string) (uint64, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "18446744073709551615" || strings.EqualFold(value, "infinity") {
		return 0, false
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func countPortConnections(port int) int {
	if port <= 0 || port > 65535 {
		return 0
	}
	count := 0
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		count += countProcNetPort(path, port)
	}
	return count
}

func countProcNetPort(path string, port int) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	portHex := fmt.Sprintf("%04X", port)
	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || fields[0] == "sl" {
			continue
		}
		if fields[3] != "01" {
			continue
		}
		_, rawPort, ok := strings.Cut(fields[1], ":")
		if !ok {
			continue
		}
		if strings.EqualFold(rawPort, portHex) {
			count++
		}
	}
	return count
}
