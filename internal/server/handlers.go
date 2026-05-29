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
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Handler struct {
	cfg config.ServerConfig
	db  *gorm.DB
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerAgentRequest struct {
	Name         string                  `json:"name"`
	Version      string                  `json:"version"`
	System       agentruntime.SystemInfo `json:"system"`
	Capabilities []string                `json:"capabilities"`
}

type heartbeatRequest struct {
	Status  string                      `json:"status"`
	Metrics agentruntime.RuntimeMetrics `json:"metrics"`
}

type createTaskRequest struct {
	AgentID string         `json:"agentId"`
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type updateTaskRequest struct {
	Status string         `json:"status"`
	Result map[string]any `json:"result"`
	Logs   string         `json:"logs"`
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
	c.JSON(http.StatusOK, gin.H{"ok": true, "version": "v0.3.0"})
}

func (h Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Username != h.cfg.AdminUser || req.Password != h.cfg.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token, err := auth.Issue(h.cfg.JWTSecret, req.Username, "panel", 12*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "expiresIn": 43200})
}

func (h Handler) overview(c *gin.Context) {
	var total int64
	var online int64
	var nodes int64
	h.db.Model(&models.Agent{}).Count(&total)
	h.db.Model(&models.Agent{}).Where("status = ?", "online").Count(&online)
	h.db.Model(&models.Node{}).Count(&nodes)
	c.JSON(http.StatusOK, gin.H{
		"agentsTotal":  total,
		"agentsOnline": online,
		"nodesTotal":   nodes,
		"version":      "v0.3.0",
	})
}

func (h Handler) listAgents(c *gin.Context) {
	var agents []models.Agent
	if err := h.db.Order("updated_at desc").Find(&agents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query agents failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": agents})
}

func (h Handler) agentInstallScript(c *gin.Context) {
	serverURL := strings.TrimRight(c.Query("serverUrl"), "/")
	if serverURL == "" {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		serverURL = fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, h.cfg.SecurePath)
	}
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -Eeuo pipefail

PANEL_URL=%q
JOIN_TOKEN=%q
AGENT_NAME="${OUUI_AGENT_NAME:-$(hostname)}"
INSTALL_DIR="${OUUI_AGENT_INSTALL_DIR:-/opt/ou-ui-agent}"
DATA_DIR="${OUUI_AGENT_DATA_DIR:-/var/lib/ou-ui-agent}"

echo "OU-UI Agent 中文一键安装脚本"
echo "主控地址：$PANEL_URL"
echo "Agent 名称：$AGENT_NAME"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "请使用 root 运行 Agent 安装脚本。" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR" "$DATA_DIR"
chmod 700 "$DATA_DIR"

cat >"$INSTALL_DIR/ou-ui-agent.env" <<EOF_ENV
OUUI_SERVER_URL=$PANEL_URL
OUUI_AGENT_JOIN_TOKEN=$JOIN_TOKEN
OUUI_AGENT_NAME=$AGENT_NAME
OUUI_AGENT_DATA_DIR=$DATA_DIR
EOF_ENV
chmod 600 "$INSTALL_DIR/ou-ui-agent.env"

echo "v0.3.0 暂未发布预编译 Agent 二进制。"
echo "请在源码目录执行：go build -o $INSTALL_DIR/ou-ui-agent ./apps/agent"

cat >/etc/systemd/system/ou-ui-agent.service <<EOF_SERVICE
[Unit]
Description=OU-UI Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$INSTALL_DIR/ou-ui-agent.env
ExecStart=$INSTALL_DIR/ou-ui-agent
Restart=always
RestartSec=5
WorkingDirectory=$DATA_DIR

[Install]
WantedBy=multi-user.target
EOF_SERVICE

systemctl daemon-reload
echo "Agent 配置已写入。二进制就绪后执行：systemctl enable --now ou-ui-agent"
`, serverURL, h.cfg.AgentJoinToken)
	c.Header("Content-Type", "text/x-shellscript; charset=utf-8")
	c.String(http.StatusOK, script)
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

	agentID := "agt_" + randomHex(8)
	agentToken := "oua_" + randomHex(24)
	capabilities, _ := json.Marshal(req.Capabilities)

	agent := models.Agent{
		ID:            agentID,
		Name:          req.Name,
		Version:       req.Version,
		Status:        "online",
		Hostname:      req.System.Hostname,
		OS:            req.System.OS,
		Arch:          req.System.Arch,
		Kernel:        req.System.Kernel,
		CPUModel:      req.System.CPUModel,
		CPUCount:      req.System.CPUCount,
		MemoryTotal:   req.System.MemoryTotal,
		SwapTotal:     req.System.SwapTotal,
		Capabilities:  datatypes.JSON(capabilities),
		AgentTokenSHA: hashSecret(agentToken),
	}
	now := time.Now()
	agent.LastSeenAt = &now

	if err := h.db.Create(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create agent failed"})
		return
	}

	h.audit("agent", "register", agent.ID, agent.Name)
	c.JSON(http.StatusOK, gin.H{"agentId": agent.ID, "agentToken": agentToken})
}

func (h Handler) agentHeartbeat(c *gin.Context) {
	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Status == "" {
		req.Status = "online"
	}
	metrics, _ := json.Marshal(req.Metrics)
	now := time.Now()

	if err := h.db.Model(&models.Agent{}).Where("id = ?", c.Param("id")).Updates(map[string]any{
		"status":       req.Status,
		"last_metrics": datatypes.JSON(metrics),
		"last_seen_at": &now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update heartbeat failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h Handler) listTasks(c *gin.Context) {
	var tasks []models.Task
	if err := h.db.Order("created_at desc").Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query tasks failed"})
		return
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
	var agent models.Agent
	if err := h.db.Select("id").Where("id = ?", req.AgentID).First(&agent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}
	payload, _ := json.Marshal(req.Payload)
	task := models.Task{
		ID:      "tsk_" + randomHex(8),
		AgentID: req.AgentID,
		Type:    req.Type,
		Status:  models.TaskStatusQueued,
		Payload: datatypes.JSON(payload),
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
	var task models.Task
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ? AND status = ?", agentID, models.TaskStatusQueued).Order("created_at asc").First(&task).Error; err != nil {
			return err
		}
		now := time.Now()
		result := tx.Model(&models.Task{}).Where("id = ? AND status = ?", task.ID, models.TaskStatusQueued).Updates(map[string]any{
			"status":     models.TaskStatusRunning,
			"attempts":   task.Attempts + 1,
			"started_at": &now,
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
	result, _ := json.Marshal(req.Result)
	updates := map[string]any{
		"status": req.Status,
		"result": datatypes.JSON(result),
		"logs":   req.Logs,
	}
	if req.Status == models.TaskStatusRunning {
		now := time.Now()
		updates["started_at"] = &now
	}
	if models.IsTerminalTaskStatus(req.Status) {
		now := time.Now()
		updates["finished_at"] = &now
	}
	var task models.Task
	if err := h.db.Where("id = ? AND agent_id = ?", c.Param("taskId"), c.Param("id")).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if err := h.db.Model(&models.Task{}).Where("id = ? AND agent_id = ?", c.Param("taskId"), c.Param("id")).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update task failed"})
		return
	}
	if task.Type == models.TaskTypeNodeDeploy && models.IsTerminalTaskStatus(req.Status) {
		var payload struct {
			NodeID string `json:"nodeId"`
		}
		if err := json.Unmarshal(task.Payload, &payload); err == nil && payload.NodeID != "" {
			nodeStatus := "deployed"
			if req.Status == models.TaskStatusFailed {
				nodeStatus = "failed"
			}
			_ = h.db.Model(&models.Node{}).Where("id = ?", payload.NodeID).Update("status", nodeStatus).Error
		}
	}
	h.audit("agent", "task.update", task.ID, req.Status)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h Handler) listNodes(c *gin.Context) {
	var nodes []models.Node
	if err := h.db.Order("updated_at desc").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query nodes failed"})
		return
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
		ID:      taskID,
		AgentID: req.AgentID,
		Type:    models.TaskTypeNodeDeploy,
		Status:  models.TaskStatusQueued,
		Payload: datatypes.JSON(taskPayload),
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

func (h Handler) audit(actor, action, target, detail string) {
	_ = h.db.Create(&models.AuditLog{Actor: actor, Action: action, Target: target, Detail: detail}).Error
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))[:n]
	}
	return hex.EncodeToString(buf)
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
