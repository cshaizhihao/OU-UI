package agentruntime

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagedNodeRegistryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	err := UpsertManagedNode(dir, ManagedNodeRef{
		NodeID:      "nod_a",
		Runtime:     "xray",
		Protocol:    "vless",
		Port:        443,
		ServiceName: "ou-ui-xray-nod_a",
		ConfigPath:  "/var/lib/ou-ui-agent/runtimes/xray/active/nod_a.json",
	})
	if err != nil {
		t.Fatalf("upsert managed node: %v", err)
	}
	err = UpsertManagedNode(dir, ManagedNodeRef{
		NodeID:      "nod_a",
		Runtime:     "xray",
		Protocol:    "vless",
		Port:        8443,
		ServiceName: "ou-ui-xray-nod_a",
	})
	if err != nil {
		t.Fatalf("replace managed node: %v", err)
	}
	nodes, err := LoadManagedNodes(dir)
	if err != nil {
		t.Fatalf("load managed nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(nodes))
	}
	if nodes[0].Port != 8443 || nodes[0].ServiceName != "ou-ui-xray-nod_a" {
		t.Fatalf("unexpected node registry entry: %+v", nodes[0])
	}
}

func TestManagedNodeRegistrySkipsMissingServiceName(t *testing.T) {
	dir := t.TempDir()
	if err := UpsertManagedNode(dir, ManagedNodeRef{
		NodeID:      "nod_missing",
		ServiceName: "<nil>",
	}); err != ErrManagedNodeMissingIdentity {
		t.Fatalf("expected missing identity error, got %v", err)
	}
	nodes, err := LoadManagedNodes(dir)
	if err != nil {
		t.Fatalf("load managed nodes: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("expected missing service name to be skipped, got %+v", nodes)
	}
}

func TestParseSystemdCounters(t *testing.T) {
	props := parseSystemdProperties("IPIngressBytes=1234\nIPEgressBytes=5678\n")
	if got, ok := parseSystemdCounter(props["IPIngressBytes"]); !ok || got != 1234 {
		t.Fatalf("unexpected ingress: %d", got)
	}
	if got, ok := parseSystemdCounter(props["IPEgressBytes"]); !ok || got != 5678 {
		t.Fatalf("unexpected egress: %d", got)
	}
	if got, ok := parseSystemdCounter("18446744073709551615"); ok || got != 0 {
		t.Fatalf("disabled counter should be unavailable zero, got %d ok=%v", got, ok)
	}
	if got, ok := parseSystemdCounter("0"); !ok || got != 0 {
		t.Fatalf("zero counter should be available, got %d ok=%v", got, ok)
	}
}

func TestCountProcNetPort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tcp")
	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:01BB 0200007F:BEEF 01 00000000:00000000 00:00000000 00000000     0        0 0 1 0000000000000000 100 0 0 10 0
   1: 0100007F:01BB 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 0 1 0000000000000000 100 0 0 10 0
   1: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 0 1 0000000000000000 100 0 0 10 0
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write proc fixture: %v", err)
	}
	if got := countProcNetPort(path, 443); got != 1 {
		t.Fatalf("expected one 443 connection, got %d", got)
	}
	if got := countProcNetPort(path, 8443); got != 0 {
		t.Fatalf("expected zero 8443 connections, got %d", got)
	}
}

func TestCollectNodeTrafficSkipsFailedCounters(t *testing.T) {
	dir := t.TempDir()
	if err := UpsertManagedNode(dir, ManagedNodeRef{
		NodeID:      "nod_a",
		ServiceName: "ou-ui-xray-nod_a",
		Port:        443,
	}); err != nil {
		t.Fatalf("upsert managed node: %v", err)
	}
	oldCounters := readSystemdIPCounters
	oldConnections := countNodePortConnections
	defer func() {
		readSystemdIPCounters = oldCounters
		countNodePortConnections = oldConnections
	}()
	countNodePortConnections = func(int) int { return 3 }
	sampler := NewSamplerWithDataDir(dir)

	readSystemdIPCounters = func(string) (uint64, uint64, bool) { return 1000, 2000, true }
	first := sampler.collectNodeTraffic(timeUnix(100))
	if len(first) != 1 || first[0].RxBytes != 1000 || first[0].TxBytes != 2000 {
		t.Fatalf("unexpected first sample: %+v", first)
	}

	readSystemdIPCounters = func(string) (uint64, uint64, bool) { return 0, 0, false }
	if failed := sampler.collectNodeTraffic(timeUnix(110)); len(failed) != 0 {
		t.Fatalf("failed counter read should not emit zero sample: %+v", failed)
	}

	readSystemdIPCounters = func(string) (uint64, uint64, bool) { return 1300, 2600, true }
	next := sampler.collectNodeTraffic(timeUnix(120))
	if len(next) != 1 {
		t.Fatalf("expected recovered sample, got %+v", next)
	}
	if next[0].RxRateBps != 15 || next[0].TxRateBps != 30 {
		t.Fatalf("failed read polluted rate baseline: %+v", next[0])
	}
}

func TestCollectNodeTrafficCounterResetDoesNotSpike(t *testing.T) {
	dir := t.TempDir()
	if err := UpsertManagedNode(dir, ManagedNodeRef{
		NodeID:      "nod_reset",
		ServiceName: "ou-ui-xray-nod_reset",
	}); err != nil {
		t.Fatalf("upsert managed node: %v", err)
	}
	oldCounters := readSystemdIPCounters
	oldConnections := countNodePortConnections
	defer func() {
		readSystemdIPCounters = oldCounters
		countNodePortConnections = oldConnections
	}()
	countNodePortConnections = func(int) int { return 0 }
	sampler := NewSamplerWithDataDir(dir)

	readSystemdIPCounters = func(string) (uint64, uint64, bool) { return 5000, 5000, true }
	_ = sampler.collectNodeTraffic(timeUnix(100))
	readSystemdIPCounters = func(string) (uint64, uint64, bool) { return 50, 80, true }
	reset := sampler.collectNodeTraffic(timeUnix(110))
	if len(reset) != 1 {
		t.Fatalf("expected reset sample, got %+v", reset)
	}
	if reset[0].RxRateBps != 0 || reset[0].TxRateBps != 0 {
		t.Fatalf("counter reset should not produce a rate spike: %+v", reset[0])
	}
}

func timeUnix(sec int64) time.Time {
	return time.Unix(sec, 0).UTC()
}
