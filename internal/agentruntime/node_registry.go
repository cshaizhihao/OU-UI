package agentruntime

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var managedNodeRegistryMu sync.Mutex

var ErrManagedNodeMissingIdentity = errors.New("managed node registry requires nodeId and serviceName")

type ManagedNodeRef struct {
	NodeID      string `json:"nodeId"`
	Name        string `json:"name,omitempty"`
	Runtime     string `json:"runtime"`
	Protocol    string `json:"protocol"`
	Port        int    `json:"port"`
	ServiceName string `json:"serviceName"`
	ConfigPath  string `json:"configPath,omitempty"`
	UpdatedAt   string `json:"updatedAt"`
}

func UpsertManagedNode(dataDir string, ref ManagedNodeRef) error {
	ref.NodeID = strings.TrimSpace(ref.NodeID)
	ref.ServiceName = strings.TrimSpace(ref.ServiceName)
	if ref.NodeID == "" || isMissingRegistryValue(ref.ServiceName) {
		return ErrManagedNodeMissingIdentity
	}
	if ref.UpdatedAt == "" {
		ref.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	managedNodeRegistryMu.Lock()
	defer managedNodeRegistryMu.Unlock()
	nodes, err := loadManagedNodes(dataDir)
	if err != nil {
		return err
	}
	replaced := false
	for i := range nodes {
		if nodes[i].NodeID == ref.NodeID {
			nodes[i] = ref
			replaced = true
			break
		}
	}
	if !replaced {
		nodes = append(nodes, ref)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeID < nodes[j].NodeID
	})
	return writeManagedNodes(dataDir, nodes)
}

func LoadManagedNodes(dataDir string) ([]ManagedNodeRef, error) {
	managedNodeRegistryMu.Lock()
	defer managedNodeRegistryMu.Unlock()
	return loadManagedNodes(dataDir)
}

func loadManagedNodes(dataDir string) ([]ManagedNodeRef, error) {
	content, err := os.ReadFile(managedNodeRegistryPath(dataDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var nodes []ManagedNodeRef
	if err := json.Unmarshal(content, &nodes); err != nil {
		return nil, err
	}
	out := make([]ManagedNodeRef, 0, len(nodes))
	for _, node := range nodes {
		node.NodeID = strings.TrimSpace(node.NodeID)
		node.ServiceName = strings.TrimSpace(node.ServiceName)
		if node.NodeID == "" || isMissingRegistryValue(node.ServiceName) {
			continue
		}
		out = append(out, node)
	}
	return out, nil
}

func writeManagedNodes(dataDir string, nodes []ManagedNodeRef) error {
	path := managedNodeRegistryPath(dataDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	content, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".managed-nodes-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return err
		}
		if retryErr := os.Rename(tmpPath, path); retryErr != nil {
			return retryErr
		}
	}
	return nil
}

func managedNodeRegistryPath(dataDir string) string {
	if dataDir == "" {
		dataDir = "/var/lib/ou-ui-agent"
	}
	return filepath.Join(dataDir, "managed-nodes.json")
}

func isMissingRegistryValue(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || value == "<nil>"
}
