package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/agentruntime"
	"github.com/cshaizhihao/OU-UI/internal/auth"
	"github.com/cshaizhihao/OU-UI/internal/config"
	"github.com/cshaizhihao/OU-UI/internal/models"
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

func (h Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "version": "v0.1.0"})
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
	h.db.Model(&models.Agent{}).Count(&total)
	h.db.Model(&models.Agent{}).Where("status = ?", "online").Count(&online)
	c.JSON(http.StatusOK, gin.H{
		"agentsTotal":  total,
		"agentsOnline": online,
		"version":      "v0.1.0",
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
	payload, _ := json.Marshal(req.Payload)
	task := models.Task{
		ID:      "tsk_" + randomHex(8),
		AgentID: req.AgentID,
		Type:    req.Type,
		Status:  "queued",
		Payload: datatypes.JSON(payload),
	}
	if err := h.db.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create task failed"})
		return
	}
	h.audit("panel", "task.create", task.ID, task.Type)
	c.JSON(http.StatusOK, task)
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
