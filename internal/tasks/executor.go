package tasks

import (
	"context"
	"encoding/json"
	"fmt"
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
	ID      string         `json:"id"`
	AgentID string         `json:"agentId"`
	Type    string         `json:"type"`
	Status  string         `json:"status"`
	Payload datatypes.JSON `json:"payload"`
	Attempts int           `json:"attempts"`
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
			"capabilities": []string{"monitoring", CapabilityTaskPolling, models.TaskTypeNoop, models.TaskTypeRuntimeStatus, tuning.CapabilityHostOptimize, "xray.render", "xray.deploy", "xray.service", "hysteria2.render", "hysteria2.deploy", "hysteria2.service"},
		}, Logs: "runtime status collected"}
	case models.TaskTypeNodeDeploy:
		return e.deployNode(task)
	case models.TaskTypeHostOptimize:
		return e.optimizeHost(task)
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
				NodeID:     payload.NodeID,
				Spec:       payload.Spec,
				DataDir:    e.DataDir,
				Revision:   revision,
				ConfigPath: applyResult.ConfigPath,
				BackupPath: applyResult.BackupPath,
				ConfigDir:   applyResult.ConfigDir,
				UnitPath:    applyResult.UnitPath,
				ServiceName: applyResult.ServiceName,
				Runner:     e.Runner,
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
	return Result{
		Status: models.TaskStatusSucceeded,
		Result: result,
		Logs:   "node deploy completed",
	}
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
		Runner: deploy.OSRunner{Timeout: 2 * time.Minute, MaxOutputBytes: 4096},
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
