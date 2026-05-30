package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/agentruntime"
	"github.com/cshaizhihao/OU-UI/internal/models"
	"github.com/cshaizhihao/OU-UI/internal/provider"
	"gorm.io/datatypes"
)

func TestRegisterManagedNodeUsesConcreteServiceName(t *testing.T) {
	dir := t.TempDir()
	executor := Executor{DataDir: dir}
	payload := deployPayload{
		NodeID: "node_a",
		Spec: provider.NodeSpec{
			Runtime:  provider.RuntimeXray,
			Protocol: "vless",
			Port:     8443,
		},
	}
	registered, err := executor.registerManagedNode(payload, provider.ApplyResult{
		ConfigPath:  "/var/lib/ou-ui-agent/runtimes/xray/active/node_a.json",
		ServiceName: "apply-service",
	}, provider.HealthResult{
		ServiceName: "health-service",
	})
	if err != nil {
		t.Fatalf("register managed node: %v", err)
	}
	if !registered {
		t.Fatal("expected managed node registration")
	}
	nodes, err := agentruntime.LoadManagedNodes(dir)
	if err != nil {
		t.Fatalf("load managed nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected one registered node, got %d", len(nodes))
	}
	if nodes[0].ServiceName != "health-service" || nodes[0].ConfigPath == "" {
		t.Fatalf("unexpected managed node entry: %+v", nodes[0])
	}
}

func TestRegisterManagedNodeSkipsMissingServiceName(t *testing.T) {
	dir := t.TempDir()
	executor := Executor{DataDir: dir}
	registered, err := executor.registerManagedNode(deployPayload{
		NodeID: "node_without_service",
		Spec: provider.NodeSpec{
			Runtime: provider.RuntimeXray,
			Port:    443,
		},
	}, provider.ApplyResult{}, provider.HealthResult{})
	if err != agentruntime.ErrManagedNodeMissingIdentity {
		t.Fatalf("expected missing identity error, got %v", err)
	}
	if registered {
		t.Fatal("expected registration to be skipped without a service name")
	}
	nodes, err := agentruntime.LoadManagedNodes(dir)
	if err != nil {
		t.Fatalf("load managed nodes: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("expected no registered nodes, got %+v", nodes)
	}
}

func TestApplyRoutingPatchesManagedXrayConfigAndRestartsService(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "runtimes", "xray", "active", "node_a.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("make config dir: %v", err)
	}
	initialConfig := `{
  "log": {"loglevel": "warning"},
  "inbounds": [{"tag": "in", "listen": "0.0.0.0", "port": 443, "protocol": "vless", "settings": {}}],
  "outbounds": [
    {"protocol": "freedom", "tag": "direct"},
    {"protocol": "blackhole", "tag": "blocked"},
    {"protocol": "vmess", "tag": "OU-Auto", "settings": {"vnext": []}}
  ]
}`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o600); err != nil {
		t.Fatalf("write xray config: %v", err)
	}
	if err := agentruntime.UpsertManagedNode(dir, agentruntime.ManagedNodeRef{
		NodeID:      "node_a",
		Runtime:     "xray",
		Protocol:    "vless",
		Port:        443,
		ConfigPath:  configPath,
		ServiceName: "ou-ui-xray-node_a",
	}); err != nil {
		t.Fatalf("register managed node: %v", err)
	}
	runner := &recordingRunner{}
	payload := datatypes.JSON(`{
  "runtime": "xray",
  "generatedAt": "2026-05-30T07:40:00Z",
  "routing": {
    "domainStrategy": "IPIfNonMatch",
    "rules": [
      {"type": "field", "domain": ["geosite:category-ads-all"], "outboundTag": "blocked"},
      {"type": "field", "ip": ["geoip:private"], "outboundTag": "direct"},
      {"type": "field", "protocol": ["bittorrent"], "outboundTag": "blocked"},
      {"type": "field", "domain": ["domain:stream.example"], "outboundTag": "OU-Auto"}
    ]
  }
}`)
	result := (Executor{DataDir: dir, Runner: runner}).applyRouting(Task{
		Type:    models.TaskTypeRoutingApply,
		Payload: payload,
	})
	if result.Status != models.TaskStatusSucceeded {
		t.Fatalf("expected routing apply success, got %+v", result)
	}
	if result.Result["patchedNodes"] != 1 {
		t.Fatalf("expected one patched node, got %+v", result.Result)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read patched config: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("patched config is not valid json: %v\n%s", err, content)
	}
	routing, ok := doc["routing"].(map[string]any)
	if !ok {
		t.Fatalf("patched config missing routing: %+v", doc)
	}
	if routing["domainStrategy"] != "IPIfNonMatch" {
		t.Fatalf("unexpected domain strategy: %+v", routing)
	}
	rules, ok := routing["rules"].([]any)
	if !ok || len(rules) != 4 {
		t.Fatalf("expected four routing rules, got %+v", routing["rules"])
	}
	outbounds, ok := doc["outbounds"].([]any)
	if !ok {
		t.Fatalf("patched config missing outbounds: %+v", doc["outbounds"])
	}
	if !hasOutboundTag(outbounds, "OU-Auto") {
		t.Fatalf("expected existing routing outbound tag to be preserved, got %+v", outbounds)
	}
	if !reflect.DeepEqual(runner.commands, [][]string{{"systemctl", "restart", "ou-ui-xray-node_a"}}) {
		t.Fatalf("unexpected runner commands: %+v", runner.commands)
	}
	if _, err := os.Stat(filepath.Join(dir, "routing", "xray-routing.json")); err != nil {
		t.Fatalf("expected canonical routing payload to be stored: %v", err)
	}
}

func TestApplyRoutingRejectsMissingProxyOutboundWithoutRestart(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "runtimes", "xray", "active", "node_missing_proxy.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("make config dir: %v", err)
	}
	initialConfig := `{"outbounds": [{"protocol": "freedom", "tag": "direct"}, {"protocol": "blackhole", "tag": "blocked"}]}`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o600); err != nil {
		t.Fatalf("write xray config: %v", err)
	}
	if err := agentruntime.UpsertManagedNode(dir, agentruntime.ManagedNodeRef{
		NodeID:      "node_missing_proxy",
		Runtime:     "xray",
		ConfigPath:  configPath,
		ServiceName: "ou-ui-xray-node_missing_proxy",
	}); err != nil {
		t.Fatalf("register managed node: %v", err)
	}
	previousRouting := []byte(`{"runtime":"xray","routing":{"rules":[{"type":"field","protocol":["bittorrent"],"outboundTag":"blocked"}]}}`)
	if err := writeFileAtomic(routingPayloadPath(dir), previousRouting, 0o600); err != nil {
		t.Fatalf("write previous routing payload: %v", err)
	}
	runner := &recordingRunner{}
	result := (Executor{DataDir: dir, Runner: runner}).applyRouting(Task{
		Type: models.TaskTypeRoutingApply,
		Payload: datatypes.JSON(`{
  "runtime": "xray",
  "routing": {"rules": [{"type": "field", "domain": ["domain:stream.example"], "outboundTag": "OU-Auto"}]}
}`),
	})
	if result.Status != models.TaskStatusFailed {
		t.Fatalf("expected routing apply failure, got %+v", result)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(content) != initialConfig {
		t.Fatalf("config should not change on rejected routing tag:\n%s", content)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("service should not restart on rejected routing tag: %+v", runner.commands)
	}
	cachedRouting, err := os.ReadFile(routingPayloadPath(dir))
	if err != nil {
		t.Fatalf("read cached routing payload: %v", err)
	}
	if string(cachedRouting) != string(previousRouting) {
		t.Fatalf("failed routing apply should restore previous cached routing:\n%s", cachedRouting)
	}
}

func TestApplyRoutingRestoresConfigWhenRestartFails(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "runtimes", "xray", "active", "node_restart_fail.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("make config dir: %v", err)
	}
	initialConfig := `{"outbounds": [{"protocol": "freedom", "tag": "direct"}, {"protocol": "blackhole", "tag": "blocked"}]}`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o600); err != nil {
		t.Fatalf("write xray config: %v", err)
	}
	if err := agentruntime.UpsertManagedNode(dir, agentruntime.ManagedNodeRef{
		NodeID:      "node_restart_fail",
		Runtime:     "xray",
		ConfigPath:  configPath,
		ServiceName: "ou-ui-xray-node_restart_fail",
	}); err != nil {
		t.Fatalf("register managed node: %v", err)
	}
	runner := &recordingRunner{runErr: errors.New("restart failed")}
	result := (Executor{DataDir: dir, Runner: runner}).applyRouting(Task{
		Type: models.TaskTypeRoutingApply,
		Payload: datatypes.JSON(`{
  "runtime": "xray",
  "routing": {"rules": [{"type": "field", "protocol": ["bittorrent"], "outboundTag": "blocked"}]}
}`),
	})
	if result.Status != models.TaskStatusFailed {
		t.Fatalf("expected restart failure, got %+v", result)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(content) != initialConfig {
		t.Fatalf("config should be restored after restart failure:\n%s", content)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("expected failed restart plus rollback restart, got %+v", runner.commands)
	}
}

func TestApplyRoutingRejectsInvalidPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload datatypes.JSON
	}{
		{name: "missing routing", payload: datatypes.JSON(`{"runtime":"xray"}`)},
		{name: "unsupported runtime", payload: datatypes.JSON(`{"runtime":"sing-box","routing":{"rules":[]}}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := (Executor{DataDir: t.TempDir(), Runner: &recordingRunner{}}).applyRouting(Task{
				Type:    models.TaskTypeRoutingApply,
				Payload: tt.payload,
			})
			if result.Status != models.TaskStatusFailed {
				t.Fatalf("expected invalid payload to fail, got %+v", result)
			}
		})
	}
}

func TestMergeCachedRoutingPreservesRoutingOnXrayRedeploy(t *testing.T) {
	dir := t.TempDir()
	payload := []byte(`{
  "runtime": "xray",
  "routing": {"rules": [{"type": "field", "protocol": ["bittorrent"], "outboundTag": "blocked"}]}
}`)
	if err := writeFileAtomic(routingPayloadPath(dir), payload, 0o600); err != nil {
		t.Fatalf("write cached routing payload: %v", err)
	}
	rendered := []byte(`{
  "inbounds": [{"tag": "in"}],
  "outbounds": [{"protocol": "freedom", "tag": "direct"}, {"protocol": "blackhole", "tag": "blocked"}]
}`)

	merged, err := (Executor{DataDir: dir}).mergeCachedRouting(provider.RuntimeXray, rendered)
	if err != nil {
		t.Fatalf("merge cached routing: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(merged, &doc); err != nil {
		t.Fatalf("merged config invalid: %v", err)
	}
	if _, ok := doc["routing"].(map[string]any); !ok {
		t.Fatalf("expected cached routing to be merged: %+v", doc)
	}
}

type recordingRunner struct {
	commands [][]string
	runErr   error
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) (provider.CommandResult, error) {
	command := append([]string{name}, args...)
	r.commands = append(r.commands, command)
	return provider.CommandResult{Command: command, Duration: time.Millisecond}, r.runErr
}

func (r *recordingRunner) LookPath(file string) (string, error) {
	return "/usr/bin/" + file, nil
}

func hasOutboundTag(outbounds []any, tag string) bool {
	for _, item := range outbounds {
		outbound, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if outbound["tag"] == tag {
			return true
		}
	}
	return false
}
