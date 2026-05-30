package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/agentruntime"
	"github.com/cshaizhihao/OU-UI/internal/deploy"
	"github.com/cshaizhihao/OU-UI/internal/models"
	"github.com/cshaizhihao/OU-UI/internal/provider"
	"github.com/cshaizhihao/OU-UI/internal/providers"
	"github.com/cshaizhihao/OU-UI/internal/tuning"
	"gorm.io/datatypes"
)

const CapabilityTaskPolling = "task-polling"

type Task struct {
	ID       string         `json:"id"`
	AgentID  string         `json:"agentId"`
	Type     string         `json:"type"`
	Status   string         `json:"status"`
	Payload  datatypes.JSON `json:"payload"`
	Attempts int            `json:"attempts"`
}

type Result struct {
	Status string         `json:"status"`
	Result map[string]any `json:"result"`
	Logs   string         `json:"logs"`
}

type Executor struct {
	DataDir  string
	Registry provider.Registry
	Runner   provider.CommandRunner
}

func NewExecutor(dataDir string) Executor {
	if dataDir == "" {
		dataDir = "/var/lib/ou-ui-agent"
	}
	return Executor{
		DataDir:  dataDir,
		Registry: providers.DefaultRegistry(),
		Runner:   deploy.OSRunner{Timeout: 20 * time.Second, MaxOutputBytes: 2048},
	}
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
			"capabilities": []string{"monitoring", CapabilityTaskPolling, models.TaskTypeNoop, models.TaskTypeRuntimeStatus, tuning.CapabilityHostOptimize, models.TaskTypeRoutingApply, "xray.render", "xray.deploy", "xray.service", "hysteria2.render", "hysteria2.deploy", "hysteria2.service"},
		}, Logs: "runtime status collected"}
	case models.TaskTypeNodeDeploy:
		return e.deployNode(task)
	case models.TaskTypeHostOptimize:
		return e.optimizeHost(task)
	case models.TaskTypeRoutingApply:
		return e.applyRouting(task)
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
	runtimeProvider, ok := e.Registry.Get(payload.Spec.Runtime)
	if !ok {
		return failed("select runtime provider", fmt.Errorf("unsupported runtime %q", payload.Spec.Runtime))
	}
	deployer, ok := runtimeProvider.(provider.DeploymentProvider)
	if !ok {
		return failed("select deployment provider", fmt.Errorf("runtime %q does not support deployment", payload.Spec.Runtime))
	}
	revision := time.Now().UTC().Format("20060102T150405Z")
	renderStarted := time.Now()
	rendered, err := runtimeProvider.Render(payload.Spec)
	if err != nil {
		return failed("render provider config", err)
	}
	stages := []provider.StageResult{
		deploy.StageOK(provider.DeployStageRender, "provider config rendered", renderStarted),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req := provider.DeployRequest{
		NodeID:   payload.NodeID,
		Spec:     payload.Spec,
		Rendered: rendered,
		DataDir:  e.DataDir,
		Revision: revision,
		Runner:   e.Runner,
	}
	result := map[string]any{
		"nodeId":   payload.NodeID,
		"runtime":  payload.Spec.Runtime,
		"protocol": payload.Spec.Protocol,
		"revision": revision,
	}
	appendStage := func(stage provider.StageResult) {
		stages = append(stages, stage)
	}
	failWithRollback := func(stageName string, err error, applyResult provider.ApplyResult) Result {
		if applyResult.ConfigPath != "" || applyResult.BackupPath != "" {
			rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer rollbackCancel()
			rollback, rollbackErr := deployer.Rollback(rollbackCtx, provider.RollbackRequest{
				NodeID:      payload.NodeID,
				Spec:        payload.Spec,
				DataDir:     e.DataDir,
				Revision:    revision,
				ConfigPath:  applyResult.ConfigPath,
				BackupPath:  applyResult.BackupPath,
				ConfigDir:   applyResult.ConfigDir,
				UnitPath:    applyResult.UnitPath,
				ServiceName: applyResult.ServiceName,
				Runner:      e.Runner,
			})
			appendStage(rollback)
			result["rollback"] = map[string]any{
				"attempted": true,
				"status":    rollback.Status,
				"error":     errorString(rollbackErr),
			}
		}
		result["stage"] = stageName
		result["error"] = err.Error()
		result["stages"] = stages
		return Result{Status: models.TaskStatusFailed, Result: result, Logs: "node deploy failed at " + stageName}
	}

	installStage, err := deployer.Install(ctx, req)
	appendStage(installStage)
	if err != nil {
		result["stages"] = stages
		result["stage"] = string(provider.DeployStageInstall)
		result["error"] = err.Error()
		return Result{Status: models.TaskStatusFailed, Result: result, Logs: "runtime install precheck failed"}
	}
	applyResult, err := deployer.ApplyConfig(ctx, req)
	appendStage(applyResult.StageResult)
	result["configPath"] = applyResult.ConfigPath
	result["configDir"] = applyResult.ConfigDir
	result["backupPath"] = applyResult.BackupPath
	result["unitPath"] = applyResult.UnitPath
	result["serviceName"] = applyResult.ServiceName
	result["serviceMode"] = applyResult.ServiceMode
	result["managedByOuui"] = applyResult.ManagedByOUUI
	result["rollbackAvailable"] = applyResult.RollbackAvailable
	if err != nil {
		return failWithRollback(string(provider.DeployStageApply), err, applyResult)
	}
	reloadStage, err := deployer.Reload(ctx, req)
	appendStage(reloadStage)
	if err != nil {
		return failWithRollback(string(provider.DeployStageReload), err, applyResult)
	}
	healthResult, err := deployer.Health(ctx, req)
	appendStage(healthResult.StageResult)
	result["health"] = healthResult
	result["serviceName"] = healthResult.ServiceName
	result["serviceStatus"] = healthResult.ServiceStatus
	result["runtimeVersion"] = healthResult.RuntimeVersion
	result["configDir"] = healthResult.ConfigDir
	result["unitPath"] = healthResult.UnitPath
	result["serviceMode"] = healthResult.ServiceMode
	result["managedByOuui"] = healthResult.ManagedByOUUI
	if err != nil {
		return failWithRollback(string(provider.DeployStageHealth), err, applyResult)
	}
	result["stages"] = stages
	trafficRegistered, registerErr := e.registerManagedNode(payload, applyResult, healthResult)
	result["trafficRegistered"] = trafficRegistered
	if registerErr != nil {
		result["trafficRegistryError"] = registerErr.Error()
		return Result{
			Status: models.TaskStatusSucceeded,
			Result: result,
			Logs:   "node deploy completed; traffic registry warning: " + registerErr.Error(),
		}
	}
	return Result{
		Status: models.TaskStatusSucceeded,
		Result: result,
		Logs:   "node deploy completed",
	}
}

func (e Executor) registerManagedNode(payload deployPayload, applyResult provider.ApplyResult, healthResult provider.HealthResult) (bool, error) {
	serviceName := firstNonEmpty(healthResult.ServiceName, applyResult.ServiceName)
	if strings.TrimSpace(payload.NodeID) == "" || strings.TrimSpace(serviceName) == "" {
		return false, agentruntime.ErrManagedNodeMissingIdentity
	}
	return true, agentruntime.UpsertManagedNode(e.DataDir, agentruntime.ManagedNodeRef{
		NodeID:      payload.NodeID,
		Name:        payload.NodeID,
		Runtime:     string(payload.Spec.Runtime),
		Protocol:    payload.Spec.Protocol,
		Port:        payload.Spec.Port,
		ServiceName: serviceName,
		ConfigPath:  applyResult.ConfigPath,
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func failed(stage string, err error) Result {
	return Result{
		Status: models.TaskStatusFailed,
		Result: map[string]any{"stage": stage, "error": err.Error()},
		Logs:   stage + ": " + err.Error(),
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (e Executor) optimizeHost(task Task) Result {
	var req tuning.Request
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return failed("decode host optimize payload", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	optimizer := tuning.Optimizer{
		Runner:  deploy.OSRunner{Timeout: 2 * time.Minute, MaxOutputBytes: 4096},
		DataDir: e.DataDir,
	}
	result, err := optimizer.Optimize(ctx, req)
	payload := map[string]any{}
	content, _ := json.Marshal(result)
	_ = json.Unmarshal(content, &payload)
	if err != nil {
		payload["error"] = err.Error()
		return Result{
			Status: models.TaskStatusFailed,
			Result: payload,
			Logs:   "host network optimization failed: " + err.Error(),
		}
	}
	return Result{
		Status: models.TaskStatusSucceeded,
		Result: payload,
		Logs:   "host network optimization completed",
	}
}

func (e Executor) applyRouting(task Task) Result {
	if len(task.Payload) == 0 {
		return failed("decode routing payload", fmt.Errorf("payload is required"))
	}
	var payload map[string]any
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return failed("decode routing payload", err)
	}
	routingDir := filepath.Join(e.DataDir, "routing")
	if err := os.MkdirAll(routingDir, 0o700); err != nil {
		return failed("prepare routing directory", err)
	}
	path := filepath.Join(routingDir, "xray-routing.json")
	content, _ := json.MarshalIndent(payload, "", "  ")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return failed("write routing config", err)
	}
	return Result{
		Status: models.TaskStatusSucceeded,
		Result: map[string]any{
			"routingPath": path,
			"runtime":     payload["runtime"],
			"appliedAt":   time.Now().UTC().Format(time.RFC3339),
		},
		Logs: "routing config stored for runtime reload",
	}
}
