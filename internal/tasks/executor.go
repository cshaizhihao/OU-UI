package tasks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/agentruntime"
	"github.com/cshaizhihao/OU-UI/internal/models"
	"github.com/cshaizhihao/OU-UI/internal/provider"
	"github.com/cshaizhihao/OU-UI/internal/providers"
	"gorm.io/datatypes"
)

const CapabilityTaskPolling = "task-polling"

type Task struct {
	ID      string         `json:"id"`
	AgentID string         `json:"agentId"`
	Type    string         `json:"type"`
	Status  string         `json:"status"`
	Payload datatypes.JSON `json:"payload"`
}

type Result struct {
	Status string         `json:"status"`
	Result map[string]any `json:"result"`
	Logs   string         `json:"logs"`
}

type Executor struct {
	DataDir  string
	Registry provider.Registry
}

func NewExecutor(dataDir string) Executor {
	if dataDir == "" {
		dataDir = "/var/lib/ou-ui-agent"
	}
	return Executor{DataDir: dataDir, Registry: providers.DefaultRegistry()}
}

func (e Executor) Execute(task Task) Result {
	switch task.Type {
	case models.TaskTypeNoop:
		return Result{
			Status: models.TaskStatusSucceeded,
			Result: map[string]any{
				"ok":          true,
				"completedAt": time.Now().UTC().Format(time.RFC3339),
			},
			Logs: "noop completed",
		}
	case models.TaskTypeRuntimeStatus:
		return Result{Status: models.TaskStatusSucceeded, Result: map[string]any{
			"metrics":      agentruntime.CollectRuntimeMetrics(),
			"capabilities": []string{"monitoring", CapabilityTaskPolling, models.TaskTypeNoop, models.TaskTypeRuntimeStatus, "xray.render", "hysteria2.render"},
		}, Logs: "runtime status collected"}
	case models.TaskTypeNodeDeploy:
		return e.deployNode(task)
	default:
		return Result{Status: models.TaskStatusFailed, Result: map[string]any{"error": "unsupported task type"}, Logs: "unsupported task type: " + task.Type}
	}
}

type deployPayload struct {
	NodeID string            `json:"nodeId"`
	Spec   provider.NodeSpec `json:"spec"`
}

func (e Executor) deployNode(task Task) Result {
	var payload deployPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return failed("decode node deploy payload", err)
	}
	if payload.NodeID == "" {
		return failed("validate node deploy payload", fmt.Errorf("nodeId is required"))
	}
	rendered, err := e.Registry.Render(payload.Spec)
	if err != nil {
		return failed("render provider config", err)
	}

	ext := "json"
	if payload.Spec.Runtime == provider.RuntimeHysteria2 {
		ext = "yaml"
	}
	dir := filepath.Join(e.DataDir, "generated", strings.ReplaceAll(string(payload.Spec.Runtime), "/", "_"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return failed("create config directory", err)
	}
	configPath := filepath.Join(dir, payload.NodeID+"."+ext)
	if err := os.WriteFile(configPath, rendered, 0o600); err != nil {
		return failed("write rendered config", err)
	}

	return Result{
		Status: models.TaskStatusSucceeded,
		Result: map[string]any{
			"nodeId":     payload.NodeID,
			"runtime":    payload.Spec.Runtime,
			"protocol":   payload.Spec.Protocol,
			"configPath": configPath,
			"renderedAt": time.Now().UTC().Format(time.RFC3339),
		},
		Logs: "rendered provider config to " + configPath,
	}
}

func failed(stage string, err error) Result {
	return Result{
		Status: models.TaskStatusFailed,
		Result: map[string]any{"stage": stage, "error": err.Error()},
		Logs:   stage + ": " + err.Error(),
	}
}
