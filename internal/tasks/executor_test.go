package tasks

import (
	"testing"

	"github.com/cshaizhihao/OU-UI/internal/agentruntime"
	"github.com/cshaizhihao/OU-UI/internal/provider"
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
