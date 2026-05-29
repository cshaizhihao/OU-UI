package server

import (
	"net/http"
	"strings"

	"github.com/cshaizhihao/OU-UI/internal/auth"
	"github.com/cshaizhihao/OU-UI/internal/config"
	"github.com/cshaizhihao/OU-UI/internal/models"
	"github.com/cshaizhihao/OU-UI/internal/providers"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(cfg config.ServerConfig, db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	h := Handler{cfg: cfg, db: db, registry: providers.DefaultRegistry()}

	r.GET("/healthz", h.health)
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"name": "OU-UI", "version": "v0.4.0", "securePath": cfg.SecurePath})
	})

	root := r.Group(cfg.SecurePath)
	root.GET("/healthz", h.health)

	api := root.Group("/api/v1")
	api.POST("/auth/login", h.login)
	api.GET("/overview", h.requirePanelAuth(), h.overview)
	api.GET("/agents", h.requirePanelAuth(), h.listAgents)
	api.GET("/agents/install-script", h.requirePanelAuth(), h.agentInstallScript)
	api.POST("/agents/register", h.requireJoinToken(), h.registerAgent)
	api.POST("/agents/:id/heartbeat", h.requireAgentAuth(), h.agentHeartbeat)
	api.GET("/agents/:id/tasks/next", h.requireAgentAuth(), h.agentNextTask)
	api.PATCH("/agents/:id/tasks/:taskId", h.requireAgentAuth(), h.agentUpdateTask)
	api.GET("/nodes", h.requirePanelAuth(), h.listNodes)
	api.POST("/nodes", h.requirePanelAuth(), h.createNode)
	api.GET("/tasks", h.requirePanelAuth(), h.listTasks)
	api.POST("/tasks", h.requirePanelAuth(), h.createTask)

	return r
}

func bearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

func (h Handler) requirePanelAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.Parse(h.cfg.JWTSecret, bearerToken(c))
		if err != nil || claims.Kind != "panel" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set("actor", claims.Subject)
		c.Next()
	}
}

func (h Handler) requireJoinToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if bearerToken(c) != h.cfg.AgentJoinToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid join token"})
			return
		}
		c.Next()
	}
}

func (h Handler) requireAgentAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		agentID := c.Param("id")
		var agent models.Agent
		tokenHash := hashSecret(bearerToken(c))
		if err := h.db.Select("agent_token_sha", "auth_status").Where("id = ?", agentID).First(&agent).Error; err != nil || tokenHash != agent.AgentTokenSHA || agent.AuthStatus == models.AgentAuthRevoked {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
			return
		}
		c.Set("agentID", agentID)
		c.Next()
	}
}
