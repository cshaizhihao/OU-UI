package tasks

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/cshaizhihao/OU-UI/internal/xray"
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
	rendered, err = e.mergeCachedRouting(payload.Spec.Runtime, rendered)
	if err != nil {
		return failed("merge cached routing", err)
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
	var applyPayload struct {
		Runtime string          `json:"runtime"`
		Routing json.RawMessage `json:"routing"`
	}
	if err := json.Unmarshal(task.Payload, &applyPayload); err != nil {
		return failed("decode routing payload", err)
	}
	runtimeName := strings.ToLower(strings.TrimSpace(applyPayload.Runtime))
	if runtimeName == "" {
		runtimeName = string(provider.RuntimeXray)
	}
	if runtimeName != string(provider.RuntimeXray) {
		return failed("validate routing payload", fmt.Errorf("unsupported routing runtime %q", applyPayload.Runtime))
	}
	if len(applyPayload.Routing) == 0 || !json.Valid(applyPayload.Routing) {
		return failed("validate routing payload", fmt.Errorf("routing is required"))
	}
	var routingDoc map[string]any
	if err := json.Unmarshal(applyPayload.Routing, &routingDoc); err != nil {
		return failed("decode xray routing", err)
	}
	routingDir := filepath.Join(e.DataDir, "routing")
	if err := os.MkdirAll(routingDir, 0o700); err != nil {
		return failed("prepare routing directory", err)
	}
	path := routingPayloadPath(e.DataDir)
	content, _ := json.MarshalIndent(payload, "", "  ")
	previousRouting, hadPreviousRouting, err := readOptionalFile(path)
	if err != nil {
		return failed("read previous routing config", err)
	}
	if err := writeFileAtomic(path, content, 0o600); err != nil {
		return failed("write routing config", err)
	}
	patchedNodes, restartedServices, err := e.patchManagedXrayRouting(routingDoc)
	if err != nil {
		_ = restoreOptionalFile(path, previousRouting, hadPreviousRouting, 0o600)
		return failed("apply xray routing", err)
	}
	return Result{
		Status: models.TaskStatusSucceeded,
		Result: map[string]any{
			"routingPath":       path,
			"runtime":           runtimeName,
			"patchedNodes":      patchedNodes,
			"restartedServices": restartedServices,
			"appliedAt":         time.Now().UTC().Format(time.RFC3339),
		},
		Logs: "routing config applied to managed xray runtimes",
	}
}

func (e Executor) patchManagedXrayRouting(routingDoc map[string]any) (int, []string, error) {
	routingContent, err := json.Marshal(routingDoc)
	if err != nil {
		return 0, nil, err
	}
	nodes, err := agentruntime.LoadManagedNodes(e.DataDir)
	if err != nil {
		return 0, nil, err
	}
	patched := 0
	restarted := []string{}
	for _, node := range nodes {
		if strings.ToLower(strings.TrimSpace(node.Runtime)) != string(provider.RuntimeXray) {
			continue
		}
		if strings.TrimSpace(node.ConfigPath) == "" || strings.TrimSpace(node.ServiceName) == "" {
			continue
		}
		previousContent, err := patchXrayConfigRouting(node.ConfigPath, routingContent)
		if err != nil {
			return patched, restarted, err
		}
		if e.Runner == nil {
			_ = writeFileAtomic(node.ConfigPath, previousContent, 0o600)
			return patched, restarted, fmt.Errorf("command runner is required to restart %s", node.ServiceName)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		if _, err := e.Runner.LookPath("systemctl"); err != nil {
			cancel()
			_ = writeFileAtomic(node.ConfigPath, previousContent, 0o600)
			return patched, restarted, err
		}
		if _, err := e.Runner.Run(ctx, "systemctl", "restart", node.ServiceName); err != nil {
			cancel()
			_ = writeFileAtomic(node.ConfigPath, previousContent, 0o600)
			rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 20*time.Second)
			_, _ = e.Runner.Run(rollbackCtx, "systemctl", "restart", node.ServiceName)
			rollbackCancel()
			return patched, restarted, err
		}
		cancel()
		patched++
		restarted = append(restarted, node.ServiceName)
	}
	return patched, restarted, nil
}

func (e Executor) mergeCachedRouting(runtime provider.Runtime, rendered []byte) ([]byte, error) {
	if runtime != provider.RuntimeXray {
		return rendered, nil
	}
	content, err := os.ReadFile(routingPayloadPath(e.DataDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return rendered, nil
		}
		return nil, err
	}
	var payload struct {
		Routing json.RawMessage `json:"routing"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, err
	}
	if len(payload.Routing) == 0 {
		return rendered, nil
	}
	return xray.MergeRoutingConfig(rendered, payload.Routing)
}

func patchXrayConfigRouting(path string, routingContent []byte) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	next, err := xray.MergeRoutingConfig(content, routingContent)
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(path, next, 0o600); err != nil {
		return nil, err
	}
	return content, nil
}

func routingPayloadPath(dataDir string) string {
	if dataDir == "" {
		dataDir = "/var/lib/ou-ui-agent"
	}
	return filepath.Join(dataDir, "routing", "xray-routing.json")
}

func readOptionalFile(path string) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return content, true, nil
}

func restoreOptionalFile(path string, content []byte, exists bool, perm os.FileMode) error {
	if !exists {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return writeFileAtomic(path, content, perm)
}

func writeFileAtomic(path string, content []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ou-ui-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
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
		return os.Rename(tmpPath, path)
	}
	return nil
}
