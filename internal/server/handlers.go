package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/agentruntime"
	"github.com/cshaizhihao/OU-UI/internal/auth"
	"github.com/cshaizhihao/OU-UI/internal/config"
	"github.com/cshaizhihao/OU-UI/internal/models"
	"github.com/cshaizhihao/OU-UI/internal/provider"
	"github.com/cshaizhihao/OU-UI/internal/tuning"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Handler struct {
	cfg      config.ServerConfig
	db       *gorm.DB
	registry provider.Registry
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerAgentRequest struct {
	InstallID    string                  `json:"installId"`
	Name         string                  `json:"name"`
	Version      string                  `json:"version"`
	System       agentruntime.SystemInfo `json:"system"`
	Capabilities []string                `json:"capabilities"`
}

type heartbeatRequest struct {
	Status    string                      `json:"status"`
	Metrics   agentruntime.RuntimeMetrics `json:"metrics"`
	LastError string                      `json:"lastError"`
}

type createTaskRequest struct {
	AgentID     string         `json:"agentId"`
	Type        string         `json:"type"`
	Payload     map[string]any `json:"payload"`
	MaxAttempts int            `json:"maxAttempts"`
}

type updateTaskRequest struct {
	Status  string         `json:"status"`
	Result  map[string]any `json:"result"`
	Logs    string         `json:"logs"`
	Attempt int            `json:"attempt"`
}

type createNodeRequest struct {
	AgentID  string         `json:"agentId"`
	Name     string         `json:"name"`
	Runtime  string         `json:"runtime"`
	Protocol string         `json:"protocol"`
	Listen   string         `json:"listen"`
	Port     int            `json:"port"`
	Settings map[string]any `json:"settings"`
}

func (h Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "version": "v6.0.0"})
}

func (h Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == h.cfg.AdminUser && req.Password == h.cfg.AdminPassword {
		token, err := auth.IssueWithOptions(h.cfg.JWTSecret, username, "panel", 12*time.Hour, auth.IssueOptions{
			Role:       "owner",
			Scopes:     []string{"*"},
			NodeAccess: []string{"*"},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "issue token failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"token":     token,
			"expiresIn": 43200,
			"user": gin.H{
				"id":       "admin",
				"username": username,
				"role":     "owner",
				"tenantId": "",
			},
		})
		return
	}
	var user models.PanelUser
	if err := h.db.Where("username = ? AND status = ?", username, "active").First(&user).Error; err != nil || user.PasswordSHA != hashSecret(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	var tenant models.Tenant
	if user.TenantID != "" {
		if err := h.db.Where("id = ? AND status = ?", user.TenantID, "active").First(&tenant).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant is disabled"})
			return
		}
	}
	nodeAccess := stringListFromJSON(user.NodeAccess)
	if len(nodeAccess) == 0 && len(tenant.NodeAccess) > 0 {
		nodeAccess = stringListFromJSON(tenant.NodeAccess)
	}
	token, err := auth.IssueWithOptions(h.cfg.JWTSecret, user.ID, "panel", 12*time.Hour, auth.IssueOptions{
		TenantID:   user.TenantID,
		Role:       user.Role,
		Scopes:     []string{"panel:read", "panel:write"},
		NodeAccess: nodeAccess,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"expiresIn": 43200,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
			"tenantId": user.TenantID,
		},
	})
}

func (h Handler) overview(c *gin.Context) {
	var agents []models.Agent
	var nodes int64
	agentQuery := h.db
	nodeQuery := h.db.Model(&models.Node{})
	if allowed, limited := nodeAccessFilter(c); limited {
		agentQuery = agentQuery.Where("id IN ?", allowed)
		nodeQuery = nodeQuery.Where("id IN ? OR agent_id IN ?", allowed, allowed)
	}
	if err := agentQuery.Find(&agents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query overview failed"})
		return
	}
	nodeQuery.Count(&nodes)
	var online int64
	for i := range agents {
		h.decorateAgent(&agents[i])
		if agents[i].Status == models.AgentStatusOnline {
			online++
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"agentsTotal":  int64(len(agents)),
		"agentsOnline": online,
		"nodesTotal":   nodes,
		"version":      "v6.0.0",
	})
}

func (h Handler) listAgents(c *gin.Context) {
	var agents []models.Agent
	query := h.db.Order("updated_at desc")
	if allowed, limited := nodeAccessFilter(c); limited {
		query = query.Where("id IN ?", allowed)
	}
	if err := query.Find(&agents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query agents failed"})
		return
	}
	for i := range agents {
		h.decorateAgent(&agents[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": agents})
}

func (h Handler) agentInstallScript(c *gin.Context) {
	c.Header("Content-Type", "text/x-shellscript; charset=utf-8")
	c.String(http.StatusOK, h.renderAgentInstallScript(c))
}

func (h Handler) renderAgentInstallScript(c *gin.Context) string {
	serverURL := strings.TrimRight(c.Query("serverUrl"), "/")
	if serverURL == "" {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		serverURL = fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, h.cfg.SecurePath)
	}

	return renderAgentInstallScriptTextCN(serverURL, h.cfg.AgentJoinToken)
}

func renderAgentInstallScriptTextCN(serverURL, joinToken string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
set -Eeuo pipefail

PANEL_URL=%q
JOIN_TOKEN=%q
AGENT_NAME="${OUUI_AGENT_NAME:-$(hostname)}"
INSTALL_DIR="${OUUI_AGENT_INSTALL_DIR:-/opt/ou-ui-agent}"
DATA_DIR="${OUUI_AGENT_DATA_DIR:-/var/lib/ou-ui-agent}"
ENV_FILE="/etc/ou-ui/agent.env"

fail() {
  printf '[错误] %%s\n' "$1" >&2
  exit 1
}

printf 'OU-UI Agent 中文一键安装脚本\n'
printf '主控地址：%%s\n' "$PANEL_URL"
printf 'Agent 名称：%%s\n' "$AGENT_NAME"

[[ "$(id -u)" -eq 0 ]] || fail "请使用 root 运行 Agent 安装脚本。"

mkdir -p "$INSTALL_DIR" "$DATA_DIR" /etc/ou-ui
chmod 700 "$DATA_DIR"

cat >"$ENV_FILE" <<EOF_ENV
OUUI_SERVER_URL=$PANEL_URL
OUUI_AGENT_JOIN_TOKEN=$JOIN_TOKEN
OUUI_AGENT_NAME=$AGENT_NAME
OUUI_AGENT_DATA_DIR=$DATA_DIR
EOF_ENV
chmod 600 "$ENV_FILE"

if [[ ! -x "$INSTALL_DIR/ou-ui-agent" ]]; then
  printf 'v0.6.0 暂未发布预编译 Agent 二进制。\n'
  printf '请先在源码目录执行：go build -o %%s/ou-ui-agent ./apps/agent\n' "$INSTALL_DIR"
fi

cat >/etc/systemd/system/ou-ui-agent.service <<EOF_SERVICE
[Unit]
Description=OU-UI Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$ENV_FILE
ExecStart=$INSTALL_DIR/ou-ui-agent
Restart=always
RestartSec=5
WorkingDirectory=$DATA_DIR

[Install]
WantedBy=multi-user.target
EOF_SERVICE

systemctl daemon-reload
printf 'Agent 配置已写入。\n'
printf '启动命令：systemctl enable --now ou-ui-agent\n'
printf '查看状态：systemctl status ou-ui-agent --no-pager\n'
printf '查看日志：journalctl -u ou-ui-agent -f\n'
`, serverURL, joinToken)
}

func (h Handler) registerAgent(c *gin.Context) {
	var req registerAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Name == "" {
		req.Name = req.System.Hostname
	}
	if req.InstallID == "" {
		req.InstallID = "ins_" + randomHex(16)
	}

	agentToken := "oua_" + randomHex(24)
	capabilities, _ := json.Marshal(req.Capabilities)
	now := time.Now()

	var existing models.Agent
	err := h.db.Where("install_id = ?", req.InstallID).First(&existing).Error
	if err == nil {
		updates := map[string]any{
			"name":            req.Name,
			"version":         req.Version,
			"status":          models.AgentStatusOnline,
			"auth_status":     models.AgentAuthActive,
			"hostname":        req.System.Hostname,
			"os":              req.System.OS,
			"arch":            req.System.Arch,
			"kernel":          req.System.Kernel,
			"cpu_model":       req.System.CPUModel,
			"cpu_count":       req.System.CPUCount,
			"memory_total":    req.System.MemoryTotal,
			"swap_total":      req.System.SwapTotal,
			"capabilities":    datatypes.JSON(capabilities),
			"last_seen_at":    &now,
			"last_error":      "",
			"agent_token_sha": hashSecret(agentToken),
		}
		if err := h.db.Model(&models.Agent{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update agent failed"})
			return
		}
		h.audit("agent", "reenroll", existing.ID, req.Name)
		c.JSON(http.StatusOK, gin.H{"agentId": existing.ID, "agentToken": agentToken, "installId": req.InstallID})
		return
	}
	if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query agent failed"})
		return
	}

	agent := models.Agent{
		ID:            "agt_" + randomHex(8),
		InstallID:     req.InstallID,
		Name:          req.Name,
		Version:       req.Version,
		Status:        models.AgentStatusOnline,
		AuthStatus:    models.AgentAuthActive,
		Hostname:      req.System.Hostname,
		OS:            req.System.OS,
		Arch:          req.System.Arch,
		Kernel:        req.System.Kernel,
		CPUModel:      req.System.CPUModel,
		CPUCount:      req.System.CPUCount,
		MemoryTotal:   req.System.MemoryTotal,
		SwapTotal:     req.System.SwapTotal,
		Capabilities:  datatypes.JSON(capabilities),
		LastSeenAt:    &now,
		AgentTokenSHA: hashSecret(agentToken),
	}

	if err := h.db.Create(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create agent failed"})
		return
	}
	h.audit("agent", "register", agent.ID, agent.Name)
	c.JSON(http.StatusOK, gin.H{"agentId": agent.ID, "agentToken": agentToken, "installId": req.InstallID})
}

func (h Handler) agentHeartbeat(c *gin.Context) {
	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	status := models.AgentStatusOnline
	if req.Status == models.AgentStatusDegraded {
		status = models.AgentStatusDegraded
	}
	metrics, _ := json.Marshal(req.Metrics)
	now := time.Now()

	if err := h.db.Model(&models.Agent{}).Where("id = ?", c.Param("id")).Updates(map[string]any{
		"status":       status,
		"last_metrics": datatypes.JSON(metrics),
		"last_seen_at": &now,
		"last_error":   strings.TrimSpace(req.LastError),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update heartbeat failed"})
		return
	}
	h.persistNodeTraffic(c.Param("id"), req.Metrics.NodeTraffic)
	h.evaluateAgentAlerts(c.Param("id"), req.Metrics, strings.TrimSpace(req.LastError))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h Handler) listTasks(c *gin.Context) {
	h.expireLeases()
	var tasks []models.Task
	query := h.db.Order("created_at desc")
	if allowed, limited := nodeAccessFilter(c); limited {
		query = query.Where("agent_id IN ?", allowed)
	}
	if err := query.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query tasks failed"})
		return
	}
	for i := range tasks {
		tasks[i].Payload = redactJSON(tasks[i].Payload)
		tasks[i].Result = redactJSON(tasks[i].Result)
	}
	c.JSON(http.StatusOK, gin.H{"items": tasks})
}

func (h Handler) createTask(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.AgentID == "" || req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if !models.IsSupportedTaskType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported task type"})
		return
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}
	requiredCapability, err := requiredCapabilityForTask(req.Type, req.Payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.dispatchableAgent(req.AgentID, requiredCapability); err != nil {
		status := http.StatusBadRequest
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	if !canAccessNode(c, req.AgentID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "agent is outside current tenant access"})
		return
	}
	if reason := h.quotaBlockReason(c); reason != "" {
		c.JSON(http.StatusForbidden, gin.H{"error": reason})
		return
	}
	payload, _ := json.Marshal(req.Payload)
	task := models.Task{
		ID:          "tsk_" + randomHex(8),
		AgentID:     req.AgentID,
		Type:        req.Type,
		Status:      models.TaskStatusQueued,
		Payload:     datatypes.JSON(payload),
		MaxAttempts: normalizeMaxAttempts(req.MaxAttempts),
	}
	if err := h.db.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create task failed"})
		return
	}
	h.audit("panel", "task.create", task.ID, task.Type)
	c.JSON(http.StatusOK, task)
}

func (h Handler) agentNextTask(c *gin.Context) {
	agentID := c.Param("id")
	h.expireLeases()
	var task models.Task
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ? AND status = ?", agentID, models.TaskStatusQueued).Order("created_at asc").First(&task).Error; err != nil {
			return err
		}
		now := time.Now()
		leaseExpiresAt := now.Add(h.cfg.TaskTimeout())
		result := tx.Model(&models.Task{}).Where("id = ? AND status = ?", task.ID, models.TaskStatusQueued).Updates(map[string]any{
			"status":           models.TaskStatusRunning,
			"attempts":         task.Attempts + 1,
			"started_at":       &now,
			"finished_at":      nil,
			"lease_expires_at": &leaseExpiresAt,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, gin.H{"task": nil})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dequeue task failed"})
		return
	}
	h.db.First(&task, "id = ?", task.ID)
	h.audit("agent", "task.dequeue", task.ID, agentID)
	c.JSON(http.StatusOK, gin.H{"task": task})
}

func (h Handler) agentUpdateTask(c *gin.Context) {
	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}
	if !models.IsAgentTaskUpdateStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported status"})
		return
	}
	if req.Attempt <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attempt is required"})
		return
	}
	result, _ := json.Marshal(req.Result)
	updates := map[string]any{
		"status":           req.Status,
		"result":           datatypes.JSON(result),
		"logs":             req.Logs,
		"lease_expires_at": nil,
	}
	if req.Status == models.TaskStatusRunning {
		now := time.Now()
		updates["started_at"] = &now
		leaseExpiresAt := now.Add(h.cfg.TaskTimeout())
		updates["lease_expires_at"] = &leaseExpiresAt
	}
	if models.IsTerminalTaskStatus(req.Status) {
		now := time.Now()
		updates["finished_at"] = &now
	}
	if req.Status == models.TaskStatusFailed {
		updates["last_error"] = extractResultError(req.Result, req.Logs)
	}
	var task models.Task
	if err := h.db.Where("id = ? AND agent_id = ?", c.Param("taskId"), c.Param("id")).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	update := h.db.Model(&models.Task{}).Where("id = ? AND agent_id = ? AND status = ? AND attempts = ?", c.Param("taskId"), c.Param("id"), models.TaskStatusRunning, req.Attempt).Updates(updates)
	if update.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update task failed"})
		return
	}
	if update.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "task state changed; update rejected"})
		return
	}
	task.Result = datatypes.JSON(result)
	task.Logs = req.Logs
	h.syncNodeStatusForTask(task, req.Status)
	h.audit("agent", "task.update", task.ID, req.Status)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h Handler) listNodes(c *gin.Context) {
	var nodes []models.Node
	query := h.db.Order("updated_at desc")
	if allowed, limited := nodeAccessFilter(c); limited {
		query = query.Where("id IN ? OR agent_id IN ?", allowed, allowed)
	}
	if err := query.Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query nodes failed"})
		return
	}
	for i := range nodes {
		nodes[i].Spec = redactJSON(nodes[i].Spec)
	}
	c.JSON(http.StatusOK, gin.H{"items": nodes})
}

func (h Handler) createNode(c *gin.Context) {
	var req createNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.AgentID == "" || req.Runtime == "" || req.Protocol == "" || req.Port == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Name == "" {
		req.Name = req.Runtime + "-" + req.Protocol
	}
	runtimeName := normalizeRuntime(req.Runtime)
	protocolName := normalizeProtocol(req.Protocol)
	if runtimeName == "" || protocolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported runtime or protocol"})
		return
	}
	spec := map[string]any{
		"runtime":  runtimeName,
		"protocol": protocolName,
		"listen":   req.Listen,
		"port":     req.Port,
		"settings": req.Settings,
	}
	nodeSpec := provider.NodeSpec{
		Runtime:  runtimeName,
		Protocol: protocolName,
		Listen:   req.Listen,
		Port:     req.Port,
		Settings: req.Settings,
	}
	runtimeProvider, ok := h.registry.Get(runtimeName)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported runtime"})
		return
	}
	if err := runtimeProvider.Validate(nodeSpec); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.dispatchableAgent(req.AgentID, requiredCapabilityForSpec(nodeSpec)); err != nil {
		status := http.StatusBadRequest
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	if !canAccessNode(c, req.AgentID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "agent is outside current tenant access"})
		return
	}
	if reason := h.quotaBlockReason(c); reason != "" {
		c.JSON(http.StatusForbidden, gin.H{"error": reason})
		return
	}
	specJSON, _ := json.Marshal(spec)
	nodeID := "nod_" + randomHex(8)
	taskID := "tsk_" + randomHex(8)
	node := models.Node{
		ID:         nodeID,
		AgentID:    req.AgentID,
		Name:       req.Name,
		Runtime:    string(runtimeName),
		Protocol:   protocolName,
		Status:     "pending",
		Spec:       datatypes.JSON(specJSON),
		LastTaskID: taskID,
	}
	taskPayload, _ := json.Marshal(map[string]any{"nodeId": nodeID, "spec": spec})
	task := models.Task{
		ID:          taskID,
		AgentID:     req.AgentID,
		Type:        models.TaskTypeNodeDeploy,
		Status:      models.TaskStatusQueued,
		Payload:     datatypes.JSON(taskPayload),
		MaxAttempts: 2,
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		return tx.Create(&task).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create node failed"})
		return
	}
	h.audit("panel", "node.create", node.ID, node.Name)
	c.JSON(http.StatusOK, gin.H{"node": node, "task": task})
}

func (h Handler) decorateAgent(agent *models.Agent) {
	if agent.AuthStatus == "" {
		agent.AuthStatus = models.AgentAuthActive
	}
	if agent.AuthStatus == models.AgentAuthRevoked {
		agent.Status = models.AgentStatusOffline
		agent.Stale = true
		return
	}
	if agent.LastSeenAt == nil {
		agent.Status = models.AgentStatusOffline
		agent.Stale = true
		return
	}
	age := time.Since(*agent.LastSeenAt)
	if age > h.cfg.AgentOfflineAfter() {
		agent.Status = models.AgentStatusOffline
		agent.Stale = true
	} else if age > h.cfg.AgentOfflineAfter()/2 && agent.Status == models.AgentStatusOnline {
		agent.Status = models.AgentStatusDegraded
	} else if agent.Status == "" {
		agent.Status = models.AgentStatusOnline
	}

	var queue int64
	_ = h.db.Model(&models.Task{}).Where("agent_id = ? AND status IN ?", agent.ID, []string{models.TaskStatusQueued, models.TaskStatusRunning}).Count(&queue).Error
	agent.QueueCount = int(queue)
}

func (h Handler) dispatchableAgent(agentID string, requiredCapability string) (models.Agent, error) {
	var agent models.Agent
	if err := h.db.Where("id = ?", agentID).First(&agent).Error; err != nil {
		return agent, err
	}
	h.decorateAgent(&agent)
	if agent.AuthStatus != "" && agent.AuthStatus != models.AgentAuthActive {
		return agent, fmt.Errorf("agent auth status is %s", agent.AuthStatus)
	}
	if agent.Status == models.AgentStatusOffline {
		return agent, fmt.Errorf("agent is offline")
	}
	if !agentHasCapability(agent, tasksCapabilityPolling) {
		return agent, fmt.Errorf("agent does not support task polling")
	}
	if requiredCapability != "" && !agentHasCapability(agent, requiredCapability) {
		return agent, fmt.Errorf("agent does not support capability %s", requiredCapability)
	}
	return agent, nil
}

func (h Handler) expireLeases() {
	now := time.Now()
	var expired []models.Task
	if err := h.db.Where("status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at < ?", models.TaskStatusRunning, now).Find(&expired).Error; err != nil {
		return
	}
	for _, task := range expired {
		maxAttempts := task.MaxAttempts
		if maxAttempts <= 0 {
			maxAttempts = 2
		}
		lastError := "task lease expired at " + now.UTC().Format(time.RFC3339)
		updates := map[string]any{
			"lease_expires_at": nil,
			"last_error":       lastError,
		}
		if task.Attempts < maxAttempts {
			updates["status"] = models.TaskStatusQueued
			updates["started_at"] = nil
		} else {
			updates["status"] = models.TaskStatusFailed
			updates["finished_at"] = &now
		}
		update := h.db.Model(&models.Task{}).Where("id = ? AND status = ?", task.ID, models.TaskStatusRunning).Updates(updates)
		if update.Error == nil && update.RowsAffected > 0 && task.Attempts >= maxAttempts {
			result, _ := json.Marshal(map[string]any{"stage": "lease", "error": lastError})
			task.Result = datatypes.JSON(result)
			task.Logs = lastError
			h.syncNodeStatusForTask(task, models.TaskStatusFailed)
		}
	}
}

func (h Handler) syncNodeStatusForTask(task models.Task, status string) {
	if task.Type != models.TaskTypeNodeDeploy || !models.IsTerminalTaskStatus(status) {
		return
	}
	var payload struct {
		NodeID string `json:"nodeId"`
	}
	if err := json.Unmarshal(task.Payload, &payload); err != nil || payload.NodeID == "" {
		return
	}
	var result map[string]any
	_ = json.Unmarshal(task.Result, &result)
	nodeStatus := "deployed"
	updates := map[string]any{
		"status": nodeStatus,
	}
	if value := stringFromResult(result, "configPath"); value != "" {
		updates["config_path"] = value
	}
	if value := stringFromResult(result, "configDir"); value != "" {
		updates["config_dir"] = value
	}
	if value := stringFromResult(result, "unitPath"); value != "" {
		updates["unit_path"] = value
	}
	if value := stringFromResult(result, "serviceMode"); value != "" {
		updates["service_mode"] = value
	}
	if value := stringFromResult(result, "serviceName"); value != "" {
		updates["service_name"] = value
	}
	if value := stringFromResult(result, "serviceStatus"); value != "" {
		updates["service_status"] = value
	}
	if value := stringFromResult(result, "runtimeVersion"); value != "" {
		updates["runtime_version"] = value
	}
	if value, ok := boolFromResult(result, "managedByOuui"); ok {
		updates["managed_by_ou_ui"] = value
	}
	if status == models.TaskStatusFailed {
		nodeStatus = "failed"
		updates["status"] = nodeStatus
		updates["last_error"] = extractResultError(result, task.Logs)
	} else {
		now := time.Now()
		updates["last_deployed_at"] = &now
		updates["last_error"] = ""
	}
	_ = h.db.Model(&models.Node{}).Where("id = ?", payload.NodeID).Updates(updates).Error
}

func normalizeMaxAttempts(value int) int {
	if value <= 0 {
		return 2
	}
	if value > 5 {
		return 5
	}
	return value
}

const tasksCapabilityPolling = "task-polling"

func requiredCapabilityForTask(taskType string, payload map[string]any) (string, error) {
	switch taskType {
	case models.TaskTypeNoop:
		return models.TaskTypeNoop, nil
	case models.TaskTypeRuntimeStatus:
		return models.TaskTypeRuntimeStatus, nil
	case models.TaskTypeNodeDeploy:
		spec, err := nodeSpecFromTaskPayload(payload)
		if err != nil {
			return "", err
		}
		capability := requiredCapabilityForSpec(spec)
		if capability == "" {
			return "", fmt.Errorf("unsupported node.deploy runtime %q", spec.Runtime)
		}
		return capability, nil
	case models.TaskTypeHostOptimize:
		return tuning.CapabilityHostOptimize, nil
	case models.TaskTypeRoutingApply:
		return "routing.apply", nil
	default:
		return "", nil
	}
}

func nodeSpecFromTaskPayload(payload map[string]any) (provider.NodeSpec, error) {
	raw, ok := payload["spec"]
	if !ok {
		return provider.NodeSpec{}, fmt.Errorf("node.deploy payload spec is required")
	}
	content, err := json.Marshal(raw)
	if err != nil {
		return provider.NodeSpec{}, err
	}
	var spec provider.NodeSpec
	if err := json.Unmarshal(content, &spec); err != nil {
		return provider.NodeSpec{}, err
	}
	if spec.Runtime == "" {
		return provider.NodeSpec{}, fmt.Errorf("node.deploy spec runtime is required")
	}
	return spec, nil
}

func requiredCapabilityForSpec(spec provider.NodeSpec) string {
	switch spec.Runtime {
	case provider.RuntimeXray:
		return "xray.deploy"
	case provider.RuntimeHysteria2:
		return "hysteria2.deploy"
	default:
		return ""
	}
}

func agentHasCapability(agent models.Agent, required string) bool {
	if required == "" {
		return true
	}
	var capabilities []string
	if err := json.Unmarshal(agent.Capabilities, &capabilities); err != nil {
		return false
	}
	for _, capability := range capabilities {
		if strings.EqualFold(strings.TrimSpace(capability), required) {
			return true
		}
	}
	return false
}

func redactJSON(raw datatypes.JSON) datatypes.JSON {
	if len(raw) == 0 {
		return raw
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	redacted, err := json.Marshal(redactValue(value))
	if err != nil {
		return raw
	}
	return datatypes.JSON(redacted)
}

func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveKey(key) {
				out[key] = "[redacted]"
				continue
			}
			out[key] = redactValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, redactValue(item))
		}
		return out
	default:
		return typed
	}
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "_", ""))
	for _, marker := range []string{"password", "secret", "token", "privatekey", "uuid"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func extractResultError(result map[string]any, logs string) string {
	if value, ok := result["error"]; ok {
		return fmt.Sprint(value)
	}
	if value, ok := result["stage"]; ok {
		return fmt.Sprint(value)
	}
	return strings.TrimSpace(logs)
}

func stringFromResult(result map[string]any, key string) string {
	if result == nil {
		return ""
	}
	value, ok := result[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func boolFromResult(result map[string]any, key string) (bool, bool) {
	if result == nil {
		return false, false
	}
	value, ok := result[key]
	if !ok {
		return false, false
	}
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		normalized := strings.ToLower(strings.TrimSpace(typed))
		switch normalized {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		}
	}
	return false, false
}

func (h Handler) audit(actor, action, target, detail string) {
	_ = h.db.Create(&models.AuditLog{Actor: actor, Action: action, Target: target, Detail: detail}).Error
}

func randomHex(n int) string {
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	seed := sha256.Sum256([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	text := hex.EncodeToString(seed[:])
	for len(text) < n*2 {
		next := sha256.Sum256([]byte(text))
		text += hex.EncodeToString(next[:])
	}
	return text[:n*2]
}

func hashSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func normalizeRuntime(value string) provider.Runtime {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "xray", "xray-core":
		return provider.RuntimeXray
	case "hysteria2", "hy2", "hysteria":
		return provider.RuntimeHysteria2
	default:
		return ""
	}
}

func normalizeProtocol(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	switch normalized {
	case "vless", "vlessreality":
		return "vless"
	case "vmess":
		return "vmess"
	case "trojan":
		return "trojan"
	case "shadowsocks", "ss":
		return "shadowsocks"
	case "hysteria2", "hy2":
		return "hysteria2"
	default:
		return ""
	}
}
