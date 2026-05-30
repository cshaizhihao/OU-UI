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
		c.JSON(http.StatusOK, gin.H{"name": "OU-UI", "version": "v6.0.0", "securePath": cfg.SecurePath})
	})

	root := r.Group(cfg.SecurePath)
	root.GET("/healthz", h.health)

	api := root.Group("/api/v1")
	api.POST("/auth/login", h.login)
	api.GET("/overview", h.requirePanelAuth(), h.overview)
	api.GET("/agents", h.requirePanelAuth(), h.listAgents)
	api.GET("/agents/install-script", h.requirePanelAuth(), h.agentInstallScript)
	api.POST("/agents/:id/network-optimization", h.requirePanelAuth(), h.createNetworkOptimization)
	api.POST("/agents/register", h.requireJoinToken(), h.registerAgent)
	api.POST("/agents/:id/heartbeat", h.requireAgentAuth(), h.agentHeartbeat)
	api.POST("/agents/:id/node-traffic", h.requireAgentAuth(), h.agentNodeTraffic)
	api.GET("/agents/:id/tasks/next", h.requireAgentAuth(), h.agentNextTask)
	api.PATCH("/agents/:id/tasks/:taskId", h.requireAgentAuth(), h.agentUpdateTask)
	api.GET("/nodes", h.requirePanelAuth(), h.listNodes)
	api.POST("/nodes", h.requirePanelAuth(), h.createNode)
	api.GET("/nodes/:id/share", h.requirePanelAuth(), h.getNodeShare)
	api.GET("/traffic/nodes", h.requirePanelAuth(), h.listNodeTraffic)
	api.GET("/traffic/nodes/:id/samples", h.requirePanelAuth(), h.listNodeTrafficSamples)
	api.GET("/routing/rules", h.requirePanelAuth(), h.listRoutingRules)
	api.POST("/routing/rules", h.requirePanelAuth(), h.createRoutingRule)
	api.PATCH("/routing/rules/:id", h.requirePanelAuth(), h.updateRoutingRule)
	api.DELETE("/routing/rules/:id", h.requirePanelAuth(), h.deleteRoutingRule)
	api.POST("/routing/apply", h.requirePanelAuth(), h.applyRouting)
	api.GET("/routing/export/xray", h.requirePanelAuth(), h.exportXrayRouting)
	api.GET("/load-balancers", h.requirePanelAuth(), h.listLoadBalancers)
	api.POST("/load-balancers", h.requirePanelAuth(), h.createLoadBalancer)
	api.GET("/load-balancers/:id/entry", h.requirePanelAuth(), h.getLoadBalancerEntry)
	api.POST("/load-balancers/:id/health", h.requirePanelAuth(), h.updateLoadBalancerHealth)
	api.PATCH("/load-balancers/:id", h.requirePanelAuth(), h.updateLoadBalancer)
	api.GET("/webhooks", h.requirePanelAuth(), h.listWebhooks)
	api.POST("/webhooks", h.requirePanelAuth(), h.createWebhook)
	api.PATCH("/webhooks/:id", h.requirePanelAuth(), h.updateWebhook)
	api.POST("/webhooks/:id/test", h.requirePanelAuth(), h.testWebhook)
	api.GET("/alerts", h.requirePanelAuth(), h.listAlerts)
	api.GET("/subscriptions", h.requirePanelAuth(), h.listSubscriptions)
	api.POST("/subscriptions", h.requirePanelAuth(), h.createSubscription)
	api.POST("/subscriptions/:id/import", h.requirePanelAuth(), h.importSubscription)
	api.GET("/subscriptions/aggregate", h.requirePanelAuth(), h.getAggregateSubscription)
	api.GET("/external-nodes", h.requirePanelAuth(), h.listExternalNodes)
	api.GET("/clash/profiles", h.requirePanelAuth(), h.listClashProfiles)
	api.POST("/clash/profiles", h.requirePanelAuth(), h.createClashProfile)
	api.GET("/clash/profiles/:id.yaml", h.requirePanelAuth(), h.getClashProfileYAML)
	api.GET("/tenants", h.requirePanelAuth(), h.listTenants)
	api.POST("/tenants", h.requirePanelAuth(), requireOwnerRole(), h.createTenant)
	api.PATCH("/tenants/:id", h.requirePanelAuth(), requireOwnerRole(), h.updateTenant)
	api.GET("/users", h.requirePanelAuth(), h.listPanelUsers)
	api.POST("/users", h.requirePanelAuth(), requireOwnerRole(), h.createPanelUser)
	api.PATCH("/users/:id", h.requirePanelAuth(), requireOwnerRole(), h.updatePanelUser)
	api.GET("/api-docs", h.requirePanelAuth(), h.apiDocs)
	api.GET("/api-keys", h.requirePanelAuth(), requireOwnerRole(), h.listAPIKeys)
	api.POST("/api-keys", h.requirePanelAuth(), requireOwnerRole(), h.createAPIKey)
	api.PATCH("/api-keys/:id", h.requirePanelAuth(), requireOwnerRole(), h.updateAPIKey)
	api.DELETE("/api-keys/:id", h.requirePanelAuth(), requireOwnerRole(), h.revokeAPIKey)
	api.GET("/copilot/incidents", h.requirePanelAuth(), h.listCopilotIncidents)
	api.POST("/copilot/ask", h.requirePanelAuth(), h.askCopilot)
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
		token := bearerToken(c)
		if h.authenticateAPIKey(c, token) {
			if c.IsAborted() {
				return
			}
			if !authorizePanelRequest(c) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "api key scope does not allow this operation"})
				return
			}
			c.Next()
			return
		}
		claims, err := auth.Parse(h.cfg.JWTSecret, token)
		if err != nil || claims.Kind != "panel" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set("actor", claims.Subject)
		c.Set("authKind", claims.Kind)
		c.Set("tenantID", claims.TenantID)
		c.Set("role", claims.Role)
		c.Set("scopes", claims.Scopes)
		c.Set("nodeAccess", claims.NodeAccess)
		if !authorizePanelRequest(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "token scope does not allow this operation"})
			return
		}
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
