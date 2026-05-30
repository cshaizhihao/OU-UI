package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/agentruntime"
	"github.com/cshaizhihao/OU-UI/internal/models"
	"github.com/cshaizhihao/OU-UI/internal/tuning"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type networkOptimizationRequest struct {
	Profile            string `json:"profile"`
	AllowKernelInstall bool   `json:"allowKernelInstall"`
	RebootPolicy       string `json:"rebootPolicy"`
	Persist            bool   `json:"persist"`
}

func (h Handler) createNetworkOptimization(c *gin.Context) {
	var req networkOptimizationRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Profile == "" {
		req.Profile = "bbr-v3"
	}
	if req.RebootPolicy == "" {
		req.RebootPolicy = "manual"
	}
	payload := map[string]any{
		"profile":            req.Profile,
		"allowKernelInstall": req.AllowKernelInstall,
		"rebootPolicy":       req.RebootPolicy,
		"persist":            req.Persist,
	}
	if _, err := h.dispatchableAgent(c.Param("id"), tuning.CapabilityHostOptimize); err != nil {
		status := http.StatusBadRequest
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	if !canAccessNode(c, c.Param("id")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "agent is outside current tenant access"})
		return
	}
	if reason := h.quotaBlockReason(c); reason != "" {
		c.JSON(http.StatusForbidden, gin.H{"error": reason})
		return
	}
	body, _ := json.Marshal(payload)
	task := models.Task{
		ID:          "tsk_" + randomHex(8),
		AgentID:     c.Param("id"),
		Type:        models.TaskTypeHostOptimize,
		Status:      models.TaskStatusQueued,
		Payload:     datatypes.JSON(body),
		MaxAttempts: 1,
	}
	if err := h.db.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create network optimization task failed"})
		return
	}
	h.audit("panel", "host.optimize", task.ID, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"task": task})
}

type nodeTrafficRequest struct {
	Samples []agentruntime.NodeTrafficMetric `json:"samples"`
}

const (
	defaultAgentTrafficRateAlertBps = 128 << 20
	defaultNodeTrafficRateAlertBps  = 64 << 20
	defaultNodeConnectionAlertCount = 10000
)

func (h Handler) agentNodeTraffic(c *gin.Context) {
	var req nodeTrafficRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	h.persistNodeTraffic(c.Param("id"), req.Samples)
	c.JSON(http.StatusOK, gin.H{"ok": true, "count": len(req.Samples)})
}

func (h Handler) persistNodeTraffic(agentID string, samples []agentruntime.NodeTrafficMetric) {
	if len(samples) == 0 {
		return
	}
	now := time.Now().UTC()
	rows := make([]models.NodeTrafficSample, 0, len(samples))
	for _, sample := range samples {
		nodeID := strings.TrimSpace(sample.NodeID)
		if nodeID == "" {
			continue
		}
		collectedAt := now
		if sample.CollectedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, sample.CollectedAt); err == nil {
				collectedAt = parsed
			}
		}
		rows = append(rows, models.NodeTrafficSample{
			NodeID:      nodeID,
			AgentID:     agentID,
			RxBytes:     sample.RxBytes,
			TxBytes:     sample.TxBytes,
			RxRateBps:   sample.RxRateBps,
			TxRateBps:   sample.TxRateBps,
			Connections: sample.Connections,
			CollectedAt: collectedAt,
		})
	}
	if len(rows) > 0 {
		_ = h.db.Create(&rows).Error
	}
}

func (h Handler) evaluateAgentAlerts(agentID string, metrics agentruntime.RuntimeMetrics, lastError string) {
	if metrics.CPUPercent >= 90 {
		h.createAlert("critical", "agent", agentID, "cpu.overload", fmt.Sprintf("Agent CPU %.1f%% exceeds critical threshold", metrics.CPUPercent), map[string]any{"cpuPercent": metrics.CPUPercent})
	} else if metrics.CPUPercent >= 80 {
		h.createAlert("warning", "agent", agentID, "cpu.high", fmt.Sprintf("Agent CPU %.1f%% exceeds warning threshold", metrics.CPUPercent), map[string]any{"cpuPercent": metrics.CPUPercent})
	}
	if lastError != "" {
		h.createAlert("warning", "agent", agentID, "agent.error", lastError, map[string]any{"lastError": lastError})
	}
	var agent models.Agent
	if err := h.db.Select("traffic_limit").Where("id = ?", agentID).First(&agent).Error; err == nil && agent.TrafficLimit > 0 {
		used := metrics.NetRxBytes + metrics.NetTxBytes
		if used >= agent.TrafficLimit {
			h.createAlert("critical", "agent", agentID, "traffic.quota.exceeded", "Agent traffic limit exceeded", map[string]any{"usedBytes": used, "limitBytes": agent.TrafficLimit})
		} else if float64(used)/float64(agent.TrafficLimit) >= 0.9 {
			h.createAlert("warning", "agent", agentID, "traffic.quota.warning", "Agent traffic limit is above 90 percent", map[string]any{"usedBytes": used, "limitBytes": agent.TrafficLimit})
		}
	}
	agentRate := metrics.NetRxRateBps + metrics.NetTxRateBps
	if agentRate >= defaultAgentTrafficRateAlertBps {
		h.createAlert("critical", "agent", agentID, "agent.traffic.overload", "Agent traffic rate is overloaded", map[string]any{
			"rxRateBps":    metrics.NetRxRateBps,
			"txRateBps":    metrics.NetTxRateBps,
			"totalRateBps": agentRate,
			"thresholdBps": defaultAgentTrafficRateAlertBps,
		})
	}
	for _, sample := range metrics.NodeTraffic {
		nodeID := strings.TrimSpace(sample.NodeID)
		if nodeID == "" {
			continue
		}
		if sample.Connections >= defaultNodeConnectionAlertCount {
			h.createAlert("warning", "node", nodeID, "node.connections.high", "Node connection count is high", map[string]any{"connections": sample.Connections, "threshold": defaultNodeConnectionAlertCount})
		}
		nodeRate := sample.RxRateBps + sample.TxRateBps
		if nodeRate >= defaultNodeTrafficRateAlertBps {
			h.createAlert("critical", "node", nodeID, "node.traffic.overload", "Node traffic rate is overloaded", map[string]any{
				"agentId":      agentID,
				"nodeName":     sample.Name,
				"rxRateBps":    sample.RxRateBps,
				"txRateBps":    sample.TxRateBps,
				"totalRateBps": nodeRate,
				"thresholdBps": defaultNodeTrafficRateAlertBps,
			})
		}
	}
}

func (h Handler) createAlert(severity, sourceType, sourceID, eventType, message string, payload map[string]any) {
	since := time.Now().UTC().Add(-10 * time.Minute)
	var recent int64
	_ = h.db.Model(&models.AlertEvent{}).
		Where("source_type = ? AND source_id = ? AND event_type = ? AND created_at >= ?", sourceType, sourceID, eventType, since).
		Count(&recent).Error
	if recent > 0 {
		return
	}
	content, _ := json.Marshal(payload)
	event := models.AlertEvent{
		ID:         "alr_" + randomHex(8),
		Severity:   severity,
		SourceType: sourceType,
		SourceID:   sourceID,
		EventType:  eventType,
		Message:    message,
		Payload:    datatypes.JSON(content),
	}
	if err := h.db.Create(&event).Error; err == nil {
		h.deliverAlert(event)
	}
}

func (h Handler) listAlerts(c *gin.Context) {
	var events []models.AlertEvent
	query := h.db.Order("created_at desc").Limit(limitFromQuery(c, 100))
	if allowed, limited := nodeAccessFilter(c); limited {
		tenantID := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "tenantID")))
		actor := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "actor")))
		query = query.Where(
			"(source_type IN ? AND source_id IN ?) OR (source_type = ? AND source_id = ?) OR (source_type = ? AND source_id = ?)",
			[]string{"agent", "node"}, allowed,
			"tenant", tenantID,
			"user", actor,
		)
	}
	if err := query.Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query alerts failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": events})
}

func (h Handler) runMaintenanceSweep() {
	h.expireLeases()
	h.markOfflineAgents()
	h.evaluateTenantQuotas()
	h.recomputeLoadBalancers()
}

func (h Handler) markOfflineAgents() {
	cutoff := time.Now().UTC().Add(-h.cfg.AgentOfflineAfter())
	var agents []models.Agent
	if err := h.db.Where("(auth_status = ? OR auth_status = '') AND (last_seen_at IS NULL OR last_seen_at < ?)", models.AgentAuthActive, cutoff).Find(&agents).Error; err != nil {
		return
	}
	for _, agent := range agents {
		if agent.AuthStatus == models.AgentAuthRevoked {
			continue
		}
		if agent.Status != models.AgentStatusOffline {
			_ = h.db.Model(&models.Agent{}).Where("id = ?", agent.ID).Update("status", models.AgentStatusOffline).Error
		}
		h.createAlert("critical", "agent", agent.ID, "agent.offline", "Agent heartbeat timed out", map[string]any{
			"lastSeenAt": agent.LastSeenAt,
			"cutoff":     cutoff.Format(time.RFC3339),
		})
	}
}

func (h Handler) evaluateTenantQuotas() {
	var tenants []models.Tenant
	if err := h.db.Where("status = ? AND (monthly_traffic_quota > 0 OR per_node_traffic_quota > 0 OR max_connections > 0)", "active").Find(&tenants).Error; err != nil {
		return
	}
	for _, tenant := range tenants {
		nodeAccess := stringListFromJSON(tenant.NodeAccess)
		used, connections, perNode := h.quotaUsage(nodeAccess)
		if tenant.MonthlyTrafficQuota > 0 {
			ratio := float64(used) / float64(tenant.MonthlyTrafficQuota)
			if used >= tenant.MonthlyTrafficQuota {
				h.createAlert("critical", "tenant", tenant.ID, "tenant.quota.exceeded", "Tenant monthly traffic quota exceeded", map[string]any{"usedBytes": used, "quotaBytes": tenant.MonthlyTrafficQuota})
			} else if ratio >= 0.9 {
				h.createAlert("warning", "tenant", tenant.ID, "tenant.quota.warning", "Tenant monthly traffic quota is above 90 percent", map[string]any{"usedBytes": used, "quotaBytes": tenant.MonthlyTrafficQuota})
			}
		}
		if tenant.MaxConnections > 0 && connections >= tenant.MaxConnections {
			h.createAlert("critical", "tenant", tenant.ID, "tenant.connections.exceeded", "Tenant connection quota exceeded", map[string]any{"connections": connections, "maxConnections": tenant.MaxConnections})
		}
		if tenant.PerNodeTrafficQuota > 0 {
			for nodeID, nodeUsed := range perNode {
				if nodeUsed >= tenant.PerNodeTrafficQuota {
					h.createAlert("critical", "tenant", tenant.ID, "tenant.node.quota.exceeded", "Tenant per-node traffic quota exceeded", map[string]any{"nodeId": nodeID, "usedBytes": nodeUsed, "quotaBytes": tenant.PerNodeTrafficQuota})
				}
			}
		}
	}
	var users []models.PanelUser
	if err := h.db.Where("status = ? AND (monthly_traffic_quota > 0 OR per_node_traffic_quota > 0 OR max_connections > 0)", "active").Find(&users).Error; err != nil {
		return
	}
	for _, user := range users {
		nodeAccess := stringListFromJSON(user.NodeAccess)
		if len(nodeAccess) == 0 && user.TenantID != "" {
			var tenant models.Tenant
			if err := h.db.Select("node_access").First(&tenant, "id = ?", user.TenantID).Error; err == nil {
				nodeAccess = stringListFromJSON(tenant.NodeAccess)
			}
		}
		used, connections, perNode := h.quotaUsage(nodeAccess)
		if user.MonthlyTrafficQuota > 0 && used >= user.MonthlyTrafficQuota {
			h.createAlert("critical", "user", user.ID, "user.quota.exceeded", "Panel user monthly traffic quota exceeded", map[string]any{"usedBytes": used, "quotaBytes": user.MonthlyTrafficQuota})
		}
		if user.MaxConnections > 0 && connections >= user.MaxConnections {
			h.createAlert("critical", "user", user.ID, "user.connections.exceeded", "Panel user connection quota exceeded", map[string]any{"connections": connections, "maxConnections": user.MaxConnections})
		}
		if user.PerNodeTrafficQuota > 0 {
			for nodeID, nodeUsed := range perNode {
				if nodeUsed >= user.PerNodeTrafficQuota {
					h.createAlert("critical", "user", user.ID, "user.node.quota.exceeded", "Panel user per-node traffic quota exceeded", map[string]any{"nodeId": nodeID, "usedBytes": nodeUsed, "quotaBytes": user.PerNodeTrafficQuota})
				}
			}
		}
	}
}

func (h Handler) recomputeLoadBalancers() {
	var groups []models.LoadBalancerGroup
	if err := h.db.Find(&groups).Error; err != nil {
		return
	}
	for _, group := range groups {
		members, err := loadBalancerMembersFromJSON(group.Members)
		if err != nil {
			continue
		}
		decision := h.loadBalancerDecision(members, group.Strategy)
		_ = h.db.Model(&models.LoadBalancerGroup{}).Where("id = ?", group.ID).Updates(map[string]any{
			"last_decision": decision,
			"status":        loadBalancerStatusFromDecision(decision),
		}).Error
	}
}

func (h Handler) listNodeTraffic(c *gin.Context) {
	type row struct {
		NodeID      string    `json:"nodeId"`
		AgentID     string    `json:"agentId"`
		RxBytes     uint64    `json:"rxBytes"`
		TxBytes     uint64    `json:"txBytes"`
		RxRateBps   uint64    `json:"rxRateBps"`
		TxRateBps   uint64    `json:"txRateBps"`
		Connections int       `json:"connections"`
		CollectedAt time.Time `json:"collectedAt"`
	}
	var samples []models.NodeTrafficSample
	query := h.db.Order("collected_at desc").Limit(500)
	if allowed, limited := nodeAccessFilter(c); limited {
		query = query.Where("node_id IN ? OR agent_id IN ?", allowed, allowed)
	}
	if err := query.Find(&samples).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query traffic failed"})
		return
	}
	latest := map[string]row{}
	for _, sample := range samples {
		if _, ok := latest[sample.NodeID]; ok {
			continue
		}
		latest[sample.NodeID] = row{
			NodeID:      sample.NodeID,
			AgentID:     sample.AgentID,
			RxBytes:     sample.RxBytes,
			TxBytes:     sample.TxBytes,
			RxRateBps:   sample.RxRateBps,
			TxRateBps:   sample.TxRateBps,
			Connections: sample.Connections,
			CollectedAt: sample.CollectedAt,
		}
	}
	items := make([]row, 0, len(latest))
	for _, item := range latest {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CollectedAt.After(items[j].CollectedAt)
	})
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h Handler) listNodeTrafficSamples(c *gin.Context) {
	var samples []models.NodeTrafficSample
	if !canAccessNode(c, c.Param("id")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "node is outside current tenant access"})
		return
	}
	if err := h.db.Where("node_id = ?", c.Param("id")).Order("collected_at desc").Limit(limitFromQuery(c, 288)).Find(&samples).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query node traffic samples failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": samples})
}

type routingRuleRequest struct {
	Name        string `json:"name"`
	Enabled     *bool  `json:"enabled"`
	Priority    int    `json:"priority"`
	RuleType    string `json:"ruleType"`
	Match       string `json:"match"`
	Protocol    string `json:"protocol"`
	Action      string `json:"action"`
	TargetTag   string `json:"targetTag"`
	Description string `json:"description"`
}

func (h Handler) listRoutingRules(c *gin.Context) {
	var rules []models.RoutingRule
	if err := h.db.Order("priority asc, created_at asc").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query routing rules failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rules})
}

func (h Handler) createRoutingRule(c *gin.Context) {
	var req routingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Match) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule := models.RoutingRule{
		ID:          "rte_" + randomHex(8),
		Name:        strings.TrimSpace(req.Name),
		Enabled:     enabled,
		Priority:    req.Priority,
		RuleType:    normalizeRoutingRuleType(req.RuleType),
		Match:       strings.TrimSpace(req.Match),
		Protocol:    strings.TrimSpace(req.Protocol),
		Action:      normalizeRoutingAction(req.Action),
		TargetTag:   strings.TrimSpace(req.TargetTag),
		Description: strings.TrimSpace(req.Description),
	}
	if rule.Priority == 0 {
		rule.Priority = 100
	}
	if rule.RuleType == "" || rule.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported routing rule type or action"})
		return
	}
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create routing rule failed"})
		return
	}
	h.audit("panel", "routing.rule.create", rule.ID, rule.Name)
	c.JSON(http.StatusOK, rule)
}

func (h Handler) updateRoutingRule(c *gin.Context) {
	var rule models.RoutingRule
	if err := h.db.First(&rule, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "routing rule not found"})
		return
	}
	var req routingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Name != "" {
		rule.Name = strings.TrimSpace(req.Name)
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Priority != 0 {
		rule.Priority = req.Priority
	}
	if req.RuleType != "" {
		rule.RuleType = normalizeRoutingRuleType(req.RuleType)
	}
	if req.Match != "" {
		rule.Match = strings.TrimSpace(req.Match)
	}
	if req.Protocol != "" {
		rule.Protocol = strings.TrimSpace(req.Protocol)
	}
	if req.Action != "" {
		rule.Action = normalizeRoutingAction(req.Action)
	}
	if req.TargetTag != "" {
		rule.TargetTag = strings.TrimSpace(req.TargetTag)
	}
	rule.Description = strings.TrimSpace(req.Description)
	if rule.RuleType == "" || rule.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported routing rule type or action"})
		return
	}
	if err := h.db.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update routing rule failed"})
		return
	}
	h.audit("panel", "routing.rule.update", rule.ID, rule.Name)
	c.JSON(http.StatusOK, rule)
}

func (h Handler) deleteRoutingRule(c *gin.Context) {
	if err := h.db.Delete(&models.RoutingRule{}, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete routing rule failed"})
		return
	}
	h.audit("panel", "routing.rule.delete", c.Param("id"), "")
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h Handler) exportXrayRouting(c *gin.Context) {
	config, err := h.xrayRoutingConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query routing rules failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"routing": config})
}

type routingApplyRequest struct {
	AgentIDs []string `json:"agentIds"`
}

func (h Handler) applyRouting(c *gin.Context) {
	var req routingApplyRequest
	_ = c.ShouldBindJSON(&req)
	if reason := h.quotaBlockReason(c); reason != "" {
		c.JSON(http.StatusForbidden, gin.H{"error": reason})
		return
	}
	config, err := h.xrayRoutingConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query routing rules failed"})
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"runtime":     "xray",
		"routing":     config,
		"generatedAt": time.Now().UTC().Format(time.RFC3339),
	})
	var agents []models.Agent
	query := h.db.Order("updated_at desc")
	if len(req.AgentIDs) > 0 {
		query = query.Where("id IN ?", compactStringList(req.AgentIDs))
	}
	if allowed, limited := nodeAccessFilter(c); limited {
		query = query.Where("id IN ?", allowed)
	}
	if err := query.Find(&agents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query agents failed"})
		return
	}
	tasks := make([]models.Task, 0, len(agents))
	for i := range agents {
		agent := agents[i]
		h.decorateAgent(&agent)
		if agent.Status == models.AgentStatusOffline || !agentHasCapability(agent, models.TaskTypeRoutingApply) {
			continue
		}
		task := models.Task{
			ID:          "tsk_" + randomHex(8),
			AgentID:     agent.ID,
			Type:        models.TaskTypeRoutingApply,
			Status:      models.TaskStatusQueued,
			Payload:     datatypes.JSON(payload),
			MaxAttempts: 2,
		}
		tasks = append(tasks, task)
	}
	if len(tasks) > 0 {
		if err := h.db.Create(&tasks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create routing apply tasks failed"})
			return
		}
	}
	h.audit("panel", "routing.apply", strconv.Itoa(len(tasks)), "")
	c.JSON(http.StatusOK, gin.H{"tasks": tasks, "count": len(tasks)})
}

func (h Handler) xrayRoutingConfig() (gin.H, error) {
	var rules []models.RoutingRule
	if err := h.db.Where("enabled = ?", true).Order("priority asc, created_at asc").Find(&rules).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		xrayRule := map[string]any{
			"type":        "field",
			"outboundTag": outboundTagForAction(rule),
		}
		switch rule.RuleType {
		case "geoip":
			xrayRule["ip"] = []string{ensurePrefix(rule.Match, "geoip:")}
		case "geosite", "ads", "domain":
			prefix := "geosite:"
			if rule.RuleType == "domain" {
				prefix = "domain:"
			}
			xrayRule["domain"] = []string{ensurePrefix(rule.Match, prefix)}
		case "protocol":
			xrayRule["protocol"] = []string{strings.TrimSpace(rule.Match)}
		case "ip":
			xrayRule["ip"] = []string{strings.TrimSpace(rule.Match)}
		}
		if rule.Protocol != "" {
			xrayRule["network"] = rule.Protocol
		}
		out = append(out, xrayRule)
	}
	return gin.H{"domainStrategy": "IPIfNonMatch", "rules": out}, nil
}

type loadBalancerRequest struct {
	Name                string           `json:"name"`
	EntryTag            string           `json:"entryTag"`
	Strategy            string           `json:"strategy"`
	Members             []map[string]any `json:"members"`
	HealthCheckInterval int              `json:"healthCheckInterval"`
}

type loadBalancerHealthRequest struct {
	Members []map[string]any `json:"members"`
}

func (h Handler) listLoadBalancers(c *gin.Context) {
	var groups []models.LoadBalancerGroup
	if err := h.db.Order("updated_at desc").Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query load balancers failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": groups})
}

func (h Handler) createLoadBalancer(c *gin.Context) {
	var req loadBalancerRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	members, err := normalizeLoadBalancerMembers(req.Members)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	strategy := normalizeLoadBalancerStrategy(req.Strategy)
	decision := h.loadBalancerDecision(members, strategy)
	group := models.LoadBalancerGroup{
		ID:                  "lbg_" + randomHex(8),
		Name:                strings.TrimSpace(req.Name),
		EntryTag:            defaultNonEmpty(req.EntryTag, "ou-ha-"+randomHex(3)),
		Strategy:            strategy,
		Members:             mustJSON(members),
		Status:              loadBalancerStatusFromDecision(decision),
		LastDecision:        decision,
		HealthCheckInterval: req.HealthCheckInterval,
	}
	if group.HealthCheckInterval <= 0 {
		group.HealthCheckInterval = 30
	}
	if err := h.db.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create load balancer failed"})
		return
	}
	h.audit("panel", "loadbalancer.create", group.ID, group.Name)
	c.JSON(http.StatusOK, group)
}

func (h Handler) updateLoadBalancer(c *gin.Context) {
	var group models.LoadBalancerGroup
	if err := h.db.First(&group, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}
	var req loadBalancerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	members, err := loadBalancerMembersFromJSON(group.Members)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load balancer members are invalid"})
		return
	}
	if req.Name != "" {
		group.Name = strings.TrimSpace(req.Name)
	}
	if req.EntryTag != "" {
		group.EntryTag = strings.TrimSpace(req.EntryTag)
	}
	if req.Strategy != "" {
		group.Strategy = normalizeLoadBalancerStrategy(req.Strategy)
	}
	if req.Members != nil {
		members, err = normalizeLoadBalancerMembers(req.Members)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		group.Members = mustJSON(members)
	}
	if req.HealthCheckInterval > 0 {
		group.HealthCheckInterval = req.HealthCheckInterval
	}
	group.LastDecision = h.loadBalancerDecision(members, group.Strategy)
	group.Status = loadBalancerStatusFromDecision(group.LastDecision)
	if err := h.db.Save(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update load balancer failed"})
		return
	}
	h.audit("panel", "loadbalancer.update", group.ID, group.Name)
	c.JSON(http.StatusOK, group)
}

func (h Handler) getLoadBalancerEntry(c *gin.Context) {
	var group models.LoadBalancerGroup
	if err := h.db.First(&group, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}
	members, err := loadBalancerMembersFromJSON(group.Members)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load balancer members are invalid"})
		return
	}
	decision := h.loadBalancerDecision(members, group.Strategy)
	status := loadBalancerStatusFromDecision(decision)
	if err := h.db.Model(&models.LoadBalancerGroup{}).Where("id = ?", group.ID).Updates(map[string]any{
		"last_decision": decision,
		"status":        status,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "persist load balancer decision failed"})
		return
	}
	decisionMap := mapFromJSON(decision)
	c.JSON(http.StatusOK, gin.H{
		"groupId":   group.ID,
		"entryTag":  group.EntryTag,
		"status":    status,
		"selected":  stringFromAny(decisionMap["selected"]),
		"member":    mapFromAny(decisionMap["member"]),
		"decision":  decisionMap,
		"entryPath": loadBalancerEntryPath(h.cfg.SecurePath, group.ID),
	})
}

func (h Handler) updateLoadBalancerHealth(c *gin.Context) {
	var group models.LoadBalancerGroup
	if err := h.db.First(&group, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}
	var req loadBalancerHealthRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Members == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	members, err := loadBalancerMembersFromJSON(group.Members)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load balancer members are invalid"})
		return
	}
	updates := map[string]map[string]any{}
	for _, update := range req.Members {
		id := loadBalancerMemberID(update)
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "health member id is required"})
			return
		}
		updates[id] = update
	}
	now := time.Now().UTC().Format(time.RFC3339)
	seen := map[string]bool{}
	for i := range members {
		id := loadBalancerMemberID(members[i])
		update, ok := updates[id]
		if !ok {
			continue
		}
		mergeLoadBalancerHealth(members[i], update, now)
		seen[id] = true
	}
	for id, update := range updates {
		if seen[id] {
			continue
		}
		next := cloneMap(update)
		mergeLoadBalancerHealth(next, update, now)
		members = append(members, next)
	}
	members, err = normalizeLoadBalancerMembers(members)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	group.Members = mustJSON(members)
	group.LastDecision = h.loadBalancerDecision(members, group.Strategy)
	group.Status = loadBalancerStatusFromDecision(group.LastDecision)
	if err := h.db.Save(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update load balancer health failed"})
		return
	}
	h.audit("panel", "loadbalancer.health", group.ID, strconv.Itoa(len(req.Members)))
	c.JSON(http.StatusOK, gin.H{"item": group, "decision": mapFromJSON(group.LastDecision)})
}

func (h Handler) loadBalancerDecision(members []map[string]any, strategies ...string) datatypes.JSON {
	strategy := "latency-loss"
	if len(strategies) > 0 {
		strategy = normalizeLoadBalancerStrategy(strategies[0])
	}
	agentStates := h.loadBalancerAgentStates(members)
	type candidate struct {
		ID      string
		Score   float64
		Member  map[string]any
		Summary map[string]any
	}
	candidates := make([]candidate, 0, len(members))
	rejected := make([]map[string]any, 0)
	for _, member := range members {
		id := loadBalancerMemberID(member)
		if id == "" {
			continue
		}
		status := normalizeLoadBalancerMemberStatus(stringFromAny(member["status"]))
		if status == "" {
			status = "healthy"
		}
		if enabled, ok := boolFromAny(member["enabled"]); ok && !enabled {
			rejected = append(rejected, gin.H{"id": id, "status": "disabled", "reason": "member disabled"})
			continue
		}
		if !loadBalancerStatusEligible(status) {
			rejected = append(rejected, gin.H{"id": id, "status": status, "reason": "member status is not eligible"})
			continue
		}
		if reason := h.loadBalancerAgentBlockReason(member, agentStates); reason != "" {
			rejected = append(rejected, gin.H{"id": id, "status": status, "reason": reason})
			continue
		}
		latency, hasLatency := numberFromAny(member["latencyMs"])
		loss, hasLoss := numberFromAny(member["lossPercent"])
		weight, hasWeight := numberFromAny(member["weight"])
		if weight <= 0 {
			weight = 1
		}
		if !hasWeight {
			weight = 1
		}
		if !hasLatency {
			latency = 1000
		}
		if !hasLoss {
			loss = 100
		}
		if latency < 0 || loss < 0 {
			rejected = append(rejected, gin.H{"id": id, "status": status, "reason": "negative metrics are invalid"})
			continue
		}
		score := (latency + loss*100) / weight
		if strategy == "weighted" {
			score = (latency + loss*100 + 100) / weight
		}
		if status == "degraded" {
			score += 250
		}
		summary := gin.H{
			"id":          id,
			"status":      status,
			"latencyMs":   latency,
			"lossPercent": loss,
			"weight":      weight,
			"score":       score,
		}
		if !hasLatency {
			summary["latencyEstimated"] = true
		}
		if !hasLoss {
			summary["lossEstimated"] = true
		}
		candidates = append(candidates, candidate{ID: id, Score: score, Member: cloneMap(member), Summary: summary})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].Score < candidates[j].Score
	})
	candidateSummaries := make([]map[string]any, 0, len(candidates))
	for _, item := range candidates {
		candidateSummaries = append(candidateSummaries, item.Summary)
	}
	payload := map[string]any{
		"selected":   "",
		"score":      0,
		"status":     "degraded",
		"strategy":   strategy,
		"member":     map[string]any{},
		"candidates": candidateSummaries,
		"rejected":   rejected,
		"decidedAt":  time.Now().UTC().Format(time.RFC3339),
	}
	if len(candidates) > 0 {
		payload["selected"] = candidates[0].ID
		payload["score"] = candidates[0].Score
		payload["status"] = "ready"
		payload["member"] = candidates[0].Member
	}
	return mustJSON(payload)
}

func loadBalancerMembersFromJSON(raw datatypes.JSON) ([]map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var members []map[string]any
	if err := json.Unmarshal(raw, &members); err != nil {
		return nil, err
	}
	return normalizeLoadBalancerMembers(members)
}

func normalizeLoadBalancerMembers(members []map[string]any) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(members))
	for _, member := range members {
		next := cloneMap(member)
		id := loadBalancerMemberID(next)
		if id == "" {
			return nil, fmt.Errorf("load balancer member id is required")
		}
		next["id"] = id
		status := normalizeLoadBalancerMemberStatus(stringFromAny(next["status"]))
		if status == "" {
			status = "healthy"
		}
		next["status"] = status
		if weight, ok := numberFromAny(next["weight"]); ok {
			if weight < 0 {
				return nil, fmt.Errorf("load balancer member %s has invalid weight", id)
			}
		} else {
			next["weight"] = 1
		}
		if latency, ok := numberFromAny(next["latencyMs"]); ok && latency < 0 {
			return nil, fmt.Errorf("load balancer member %s has invalid latency", id)
		}
		if loss, ok := numberFromAny(next["lossPercent"]); ok && loss < 0 {
			return nil, fmt.Errorf("load balancer member %s has invalid loss", id)
		}
		if port, ok := numberFromAny(next["port"]); ok && (port < 0 || port > 65535) {
			return nil, fmt.Errorf("load balancer member %s has invalid port", id)
		}
		out = append(out, next)
	}
	return out, nil
}

func normalizeLoadBalancerStrategy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "weighted", "latency-loss":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "latency-loss"
	}
}

func normalizeLoadBalancerMemberStatus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func loadBalancerStatusEligible(status string) bool {
	switch normalizeLoadBalancerMemberStatus(status) {
	case "", "healthy", "online", "ready", "active", "degraded":
		return true
	case "down", "offline", "disabled", "revoked", "failed", "failing", "inactive", "maintenance":
		return false
	default:
		return false
	}
}

func loadBalancerStatusFromDecision(decision datatypes.JSON) string {
	payload := mapFromJSON(decision)
	if stringFromAny(payload["selected"]) == "" {
		return "degraded"
	}
	if status := stringFromAny(payload["status"]); status != "" {
		return status
	}
	return "ready"
}

func loadBalancerMemberID(member map[string]any) string {
	for _, key := range []string{"id", "nodeId", "agentId"} {
		if value := stringFromAny(member[key]); value != "" {
			return value
		}
	}
	return ""
}

func loadBalancerMemberAgentID(member map[string]any) string {
	if value := stringFromAny(member["agentId"]); value != "" {
		return value
	}
	if value := stringFromAny(member["id"]); value != "" && stringFromAny(member["nodeId"]) == "" {
		return value
	}
	return ""
}

func (h Handler) loadBalancerAgentStates(members []map[string]any) map[string]models.Agent {
	out := map[string]models.Agent{}
	if h.db == nil {
		return out
	}
	ids := make([]string, 0, len(members))
	seen := map[string]bool{}
	for _, member := range members {
		id := loadBalancerMemberAgentID(member)
		if id == "" || seen[id] {
			continue
		}
		ids = append(ids, id)
		seen[id] = true
	}
	if len(ids) == 0 {
		return out
	}
	var agents []models.Agent
	if err := h.db.Where("id IN ?", ids).Find(&agents).Error; err != nil {
		return out
	}
	for _, agent := range agents {
		out[agent.ID] = agent
	}
	return out
}

func (h Handler) loadBalancerAgentBlockReason(member map[string]any, states map[string]models.Agent) string {
	agentID := loadBalancerMemberAgentID(member)
	if agentID == "" {
		return ""
	}
	agent, ok := states[agentID]
	if !ok {
		return ""
	}
	if agent.AuthStatus == models.AgentAuthRevoked {
		return "agent revoked"
	}
	if agent.Status == models.AgentStatusOffline {
		return "agent offline"
	}
	if agent.LastSeenAt != nil && h.cfg.AgentOfflineAfter() > 0 && agent.LastSeenAt.Before(time.Now().UTC().Add(-h.cfg.AgentOfflineAfter())) {
		return "agent heartbeat stale"
	}
	return ""
}

func mergeLoadBalancerHealth(member map[string]any, update map[string]any, checkedAt string) {
	for _, key := range []string{"agentId", "nodeId", "name", "address", "port", "status", "enabled", "latencyMs", "lossPercent", "weight", "lastError", "lastCheckedAt"} {
		if value, ok := update[key]; ok {
			member[key] = value
		}
	}
	if stringFromAny(member["lastCheckedAt"]) == "" {
		member["lastCheckedAt"] = checkedAt
	}
}

func loadBalancerEntryPath(securePath, groupID string) string {
	base := strings.TrimRight(defaultNonEmpty(securePath, "/ou-ui"), "/")
	return fmt.Sprintf("%s/api/v1/load-balancers/%s/entry", base, groupID)
}

type webhookRequest struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	URL        string   `json:"url"`
	Secret     string   `json:"secret"`
	ChatID     string   `json:"chatId"`
	Enabled    *bool    `json:"enabled"`
	EventTypes []string `json:"eventTypes"`
}

func (h Handler) listWebhooks(c *gin.Context) {
	var hooks []models.WebhookEndpoint
	if err := h.db.Order("updated_at desc").Find(&hooks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query webhooks failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": hooks})
}

func (h Handler) createWebhook(c *gin.Context) {
	var req webhookRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	hook := models.WebhookEndpoint{
		ID:         "whk_" + randomHex(8),
		Name:       strings.TrimSpace(req.Name),
		Kind:       normalizeWebhookKind(req.Kind),
		URL:        strings.TrimSpace(req.URL),
		Secret:     strings.TrimSpace(req.Secret),
		ChatID:     strings.TrimSpace(req.ChatID),
		Enabled:    enabled,
		EventTypes: mustJSON(compactStringList(req.EventTypes)),
	}
	if err := validateWebhookConfig(hook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Create(&hook).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create webhook failed"})
		return
	}
	h.audit("panel", "webhook.create", hook.ID, hook.Name)
	c.JSON(http.StatusOK, hook)
}

func (h Handler) updateWebhook(c *gin.Context) {
	var hook models.WebhookEndpoint
	if err := h.db.First(&hook, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}
	var req webhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Name != "" {
		hook.Name = strings.TrimSpace(req.Name)
	}
	if req.Kind != "" {
		hook.Kind = normalizeWebhookKind(req.Kind)
	}
	if req.URL != "" {
		hook.URL = strings.TrimSpace(req.URL)
	}
	if req.Secret != "" {
		hook.Secret = strings.TrimSpace(req.Secret)
	}
	if req.ChatID != "" {
		hook.ChatID = strings.TrimSpace(req.ChatID)
	}
	if req.Enabled != nil {
		hook.Enabled = *req.Enabled
	}
	if req.EventTypes != nil {
		hook.EventTypes = mustJSON(compactStringList(req.EventTypes))
	}
	if err := validateWebhookConfig(hook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Save(&hook).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update webhook failed"})
		return
	}
	h.audit("panel", "webhook.update", hook.ID, hook.Name)
	c.JSON(http.StatusOK, hook)
}

func (h Handler) testWebhook(c *gin.Context) {
	var hook models.WebhookEndpoint
	if err := h.db.First(&hook, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}
	event := models.AlertEvent{
		ID:         "alr_" + randomHex(8),
		Severity:   "info",
		SourceType: "system",
		SourceID:   "ou-ui",
		EventType:  "webhook.test",
		Message:    "OU-UI webhook test event",
		Payload:    mustJSON(map[string]any{"test": true}),
	}
	err := deliverWebhook(hook, event)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h Handler) deliverAlert(event models.AlertEvent) {
	var hooks []models.WebhookEndpoint
	if err := h.db.Where("enabled = ?", true).Find(&hooks).Error; err != nil {
		return
	}
	delivered := false
	lastError := ""
	for _, hook := range hooks {
		if !webhookAccepts(hook, event.EventType) {
			continue
		}
		if err := deliverWebhook(hook, event); err != nil {
			lastError = err.Error()
			continue
		}
		delivered = true
	}
	updates := map[string]any{"delivered": delivered, "last_error": lastError}
	if delivered {
		now := time.Now().UTC()
		updates["delivered_at"] = &now
	}
	_ = h.db.Model(&models.AlertEvent{}).Where("id = ?", event.ID).Updates(updates).Error
}

func webhookAccepts(hook models.WebhookEndpoint, eventType string) bool {
	var eventTypes []string
	if err := json.Unmarshal(hook.EventTypes, &eventTypes); err != nil || len(eventTypes) == 0 {
		return true
	}
	for _, item := range eventTypes {
		if item == "*" || strings.EqualFold(strings.TrimSpace(item), eventType) {
			return true
		}
	}
	return false
}

func normalizeWebhookKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "telegram":
		return "telegram"
	case "serverchan", "server-chan", "server_chan":
		return "serverchan"
	case "", "generic", "webhook":
		return "generic"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func validateWebhookConfig(hook models.WebhookEndpoint) error {
	switch normalizeWebhookKind(hook.Kind) {
	case "telegram":
		if strings.TrimSpace(hook.Secret) == "" || strings.TrimSpace(hook.ChatID) == "" {
			return fmt.Errorf("telegram token and chatId are required")
		}
	case "serverchan":
		if strings.TrimSpace(hook.URL) == "" {
			return fmt.Errorf("serverchan url is required")
		}
	default:
		if strings.TrimSpace(hook.URL) == "" {
			return fmt.Errorf("webhook url is required")
		}
	}
	return nil
}

func deliverWebhook(hook models.WebhookEndpoint, event models.AlertEvent) error {
	client := &http.Client{Timeout: 6 * time.Second}
	message := fmt.Sprintf("[%s] %s: %s", strings.ToUpper(event.Severity), event.EventType, event.Message)
	switch normalizeWebhookKind(hook.Kind) {
	case "telegram":
		token := strings.TrimSpace(hook.Secret)
		if token == "" || hook.ChatID == "" {
			return fmt.Errorf("telegram token and chatId are required")
		}
		endpoint := "https://api.telegram.org/bot" + token + "/sendMessage"
		if strings.TrimSpace(hook.URL) != "" {
			endpoint = strings.TrimSpace(hook.URL)
		}
		payload := map[string]any{"chat_id": hook.ChatID, "text": message}
		return postJSON(client, endpoint, payload, "")
	case "serverchan":
		if hook.URL == "" {
			return fmt.Errorf("serverchan url is required")
		}
		payload := map[string]any{"title": "OU-UI Alert", "desp": message}
		return postJSON(client, hook.URL, payload, "")
	default:
		if hook.URL == "" {
			return fmt.Errorf("webhook url is required")
		}
		payload := map[string]any{"event": event, "message": message}
		return postJSON(client, hook.URL, payload, hook.Secret)
	}
}

type subscriptionRequest struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Format  string `json:"format"`
	Enabled *bool  `json:"enabled"`
	Content string `json:"content"`
}

func (h Handler) listSubscriptions(c *gin.Context) {
	var subscriptions []models.ExternalSubscription
	if err := h.db.Order("updated_at desc").Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query subscriptions failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": subscriptions})
}

func (h Handler) createSubscription(c *gin.Context) {
	var req subscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	sub := models.ExternalSubscription{
		ID:      "sub_" + randomHex(8),
		Name:    strings.TrimSpace(req.Name),
		URL:     strings.TrimSpace(req.URL),
		Format:  defaultNonEmpty(req.Format, "auto"),
		Enabled: enabled,
	}
	if err := h.db.Create(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create subscription failed"})
		return
	}
	if req.Content != "" || req.URL != "" {
		_, _ = h.importSubscriptionContent(&sub, req.Content)
	}
	h.audit("panel", "subscription.create", sub.ID, sub.Name)
	c.JSON(http.StatusOK, sub)
}

func (h Handler) importSubscription(c *gin.Context) {
	var sub models.ExternalSubscription
	if err := h.db.First(&sub, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	var req subscriptionRequest
	_ = c.ShouldBindJSON(&req)
	nodes, err := h.importSubscriptionContent(&sub, req.Content)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"subscription": sub, "imported": len(nodes), "items": nodes})
}

func (h Handler) importSubscriptionContent(sub *models.ExternalSubscription, content string) ([]models.ExternalNode, error) {
	if content == "" {
		if sub.URL == "" {
			return nil, fmt.Errorf("subscription url or content is required")
		}
		fetched, err := fetchText(sub.URL)
		if err != nil {
			_ = h.db.Model(sub).Updates(map[string]any{"last_error": err.Error()}).Error
			return nil, err
		}
		content = fetched
	}
	nodes := parseExternalNodes(sub.ID, content)
	now := time.Now().UTC()
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.ExternalNode{}, "subscription_id = ?", sub.ID).Error; err != nil {
			return err
		}
		if len(nodes) > 0 {
			if err := tx.Create(&nodes).Error; err != nil {
				return err
			}
		}
		return tx.Model(sub).Updates(map[string]any{"last_fetched_at": &now, "last_error": ""}).Error
	})
	return nodes, err
}

func (h Handler) listExternalNodes(c *gin.Context) {
	var nodes []models.ExternalNode
	query := h.db.Order("updated_at desc")
	if subID := strings.TrimSpace(c.Query("subscriptionId")); subID != "" {
		query = query.Where("subscription_id = ?", subID)
	}
	if err := query.Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query external nodes failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": nodes})
}

func (h Handler) getAggregateSubscription(c *gin.Context) {
	format := strings.ToLower(strings.TrimSpace(c.Query("format")))
	if format == "" {
		format = "clash"
	}
	switch format {
	case "clash", "yaml", "yml":
		c.Header("Content-Type", "text/yaml; charset=utf-8")
		c.Header("Content-Disposition", `inline; filename="ou-ui-aggregate.yaml"`)
		c.String(http.StatusOK, h.generateClashYAML(clashProfileRequest{Name: "OU-UI Aggregate"}))
	case "v2ray", "v2rayn", "share", "base64":
		content := h.generateV2RaySubscription()
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Content-Disposition", `inline; filename="ou-ui-aggregate.txt"`)
		c.String(http.StatusOK, base64.StdEncoding.EncodeToString([]byte(content)))
	case "raw":
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Content-Disposition", `inline; filename="ou-ui-aggregate-raw.txt"`)
		c.String(http.StatusOK, h.generateV2RaySubscription())
	case "sing-box", "singbox", "json":
		content, err := json.MarshalIndent(h.generateSingBoxConfig(), "", "  ")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "generate sing-box config failed"})
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("Content-Disposition", `inline; filename="ou-ui-aggregate-sing-box.json"`)
		c.String(http.StatusOK, string(content))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported aggregate format"})
	}
}

type clashProfileRequest struct {
	Name          string           `json:"name"`
	RuleProviders []map[string]any `json:"ruleProviders"`
	ProxyGroups   []map[string]any `json:"proxyGroups"`
	RoutingRules  []string         `json:"routingRules"`
	SelectedNodes []string         `json:"selectedNodes"`
}

func (h Handler) listClashProfiles(c *gin.Context) {
	var profiles []models.ClashProfile
	if err := h.db.Order("updated_at desc").Find(&profiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query clash profiles failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": profiles})
}

func (h Handler) createClashProfile(c *gin.Context) {
	var req clashProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	yaml := h.generateClashYAML(req)
	profile := models.ClashProfile{
		ID:            "cpr_" + randomHex(8),
		Name:          strings.TrimSpace(req.Name),
		RuleProviders: mustJSON(req.RuleProviders),
		ProxyGroups:   mustJSON(req.ProxyGroups),
		RoutingRules:  mustJSON(req.RoutingRules),
		GeneratedYAML: yaml,
	}
	if err := h.db.Create(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create clash profile failed"})
		return
	}
	h.audit("panel", "clash.profile.create", profile.ID, profile.Name)
	c.JSON(http.StatusOK, profile)
}

func (h Handler) getClashProfileYAML(c *gin.Context) {
	var profile models.ClashProfile
	if err := h.db.First(&profile, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "clash profile not found"})
		return
	}
	c.Header("Content-Type", "text/yaml; charset=utf-8")
	c.String(http.StatusOK, profile.GeneratedYAML)
}

func (h Handler) generateClashYAML(req clashProfileRequest) string {
	proxies := h.clashProxies()
	proxies = filterClashProxies(proxies, req.SelectedNodes)
	proxyNames := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		if name, ok := proxy["name"].(string); ok && name != "" {
			proxyNames = append(proxyNames, name)
		}
	}
	if len(proxyNames) == 0 {
		proxyNames = []string{"DIRECT"}
	}
	groups := normalizeClashProxyGroups(req.ProxyGroups, proxyNames)
	defaultTarget := firstClashGroupName(groups, "OU-Auto")
	rules := req.RoutingRules
	if len(rules) == 0 {
		for _, provider := range req.RuleProviders {
			if name := strings.TrimSpace(fmt.Sprint(provider["name"])); name != "" {
				rules = append(rules, "RULE-SET,"+name+","+defaultTarget)
			}
		}
		var dbRules []models.RoutingRule
		_ = h.db.Where("enabled = ?", true).Order("priority asc").Find(&dbRules).Error
		for _, rule := range dbRules {
			rules = append(rules, clashRuleLineWithTarget(rule, defaultTarget))
		}
		rules = append(rules, "MATCH,"+defaultTarget)
	}
	document := map[string]any{
		"mixed-port":          7890,
		"allow-lan":           true,
		"mode":                "rule",
		"log-level":           "warning",
		"external-controller": "127.0.0.1:9090",
		"proxies":             sanitizeClashProxies(proxies),
		"proxy-groups":        groups,
		"rules":               rules,
	}
	if len(req.RuleProviders) > 0 {
		providers := map[string]any{}
		for _, provider := range cloneMapSlice(req.RuleProviders) {
			name := strings.TrimSpace(fmt.Sprint(provider["name"]))
			if name == "" {
				name = "provider-" + randomHex(3)
			}
			delete(provider, "name")
			providers[name] = provider
		}
		document["rule-providers"] = providers
	}
	content, err := yaml.Marshal(document)
	if err != nil {
		return ""
	}
	return string(content)
}

func (h Handler) clashProxies() []map[string]any {
	var proxies []map[string]any
	var external []models.ExternalNode
	_ = h.db.Where("enabled = ?", true).Order("updated_at desc").Find(&external).Error
	for _, node := range external {
		proxy := mapFromJSON(node.Config)
		if len(proxy) == 0 {
			proxy = map[string]any{"type": node.Protocol, "server": node.Address, "port": node.Port}
		}
		proxy["name"] = node.Name
		proxy["type"] = clashType(node.Protocol)
		proxy["server"] = node.Address
		proxy["port"] = node.Port
		proxy["_nodeId"] = node.ID
		proxy["_source"] = "external"
		proxies = append(proxies, proxy)
	}
	var nodes []models.Node
	_ = h.db.Where("status <> ?", "failed").Order("updated_at desc").Find(&nodes).Error
	agentIPs := map[string]string{}
	for _, node := range nodes {
		var spec map[string]any
		_ = json.Unmarshal(node.Spec, &spec)
		settings, _ := spec["settings"].(map[string]any)
		server := agentIPs[node.AgentID]
		if server == "" {
			var agent models.Agent
			if err := h.db.Select("public_ip").First(&agent, "id = ?", node.AgentID).Error; err == nil {
				server = agent.PublicIP
				agentIPs[node.AgentID] = server
			}
		}
		if server == "" {
			server = "127.0.0.1"
		}
		protocol := strings.ToLower(fmt.Sprint(spec["protocol"]))
		proxy := map[string]any{
			"name":    node.Name,
			"type":    clashType(protocol),
			"server":  server,
			"port":    intFromAny(spec["port"]),
			"_nodeId": node.ID,
			"_source": "managed",
		}
		if uuid := firstMapString(settings, "uuid", "id"); uuid != "" {
			proxy["uuid"] = uuid
		}
		if password := firstMapString(settings, "password"); password != "" {
			proxy["password"] = password
		}
		if method := firstMapString(settings, "method"); method != "" {
			proxy["cipher"] = method
		}
		proxies = append(proxies, proxy)
	}
	return proxies
}

func filterClashProxies(proxies []map[string]any, selected []string) []map[string]any {
	selected = compactStringList(selected)
	if len(selected) == 0 || containsWildcard(selected) {
		return proxies
	}
	allowed := map[string]struct{}{}
	for _, value := range selected {
		allowed[strings.ToLower(strings.TrimSpace(value))] = struct{}{}
	}
	filtered := make([]map[string]any, 0, len(proxies))
	for _, proxy := range proxies {
		keys := []string{
			strings.ToLower(proxyString(proxy, "name")),
			strings.ToLower(proxyString(proxy, "_nodeId", "id")),
		}
		for _, key := range keys {
			if _, ok := allowed[key]; ok && key != "" {
				filtered = append(filtered, proxy)
				break
			}
		}
	}
	return filtered
}

func sanitizeClashProxies(proxies []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(proxies))
	for _, proxy := range proxies {
		next := cloneMap(proxy)
		for key := range next {
			if strings.HasPrefix(key, "_") {
				delete(next, key)
			}
		}
		out = append(out, next)
	}
	return out
}

func normalizeClashProxyGroups(groups []map[string]any, proxyNames []string) []map[string]any {
	normalized := cloneMapSlice(groups)
	if len(normalized) == 0 {
		return []map[string]any{
			{"name": "OU-Auto", "type": "url-test", "url": "https://www.gstatic.com/generate_204", "interval": 300, "proxies": proxyNames},
			{"name": "OU-Fallback", "type": "fallback", "url": "https://www.gstatic.com/generate_204", "interval": 300, "proxies": proxyNames},
		}
	}
	for index, group := range normalized {
		if strings.TrimSpace(fmt.Sprint(group["name"])) == "" {
			group["name"] = "OU-Group-" + strconv.Itoa(index+1)
		}
		groupType := strings.ToLower(strings.TrimSpace(fmt.Sprint(group["type"])))
		if groupType == "" {
			groupType = "select"
			group["type"] = groupType
		}
		groupProxies := stringSliceFromAny(group["proxies"])
		if len(groupProxies) == 0 || containsWildcard(groupProxies) {
			group["proxies"] = proxyNames
		}
		if (groupType == "url-test" || groupType == "fallback" || groupType == "load-balance") && strings.TrimSpace(fmt.Sprint(group["url"])) == "" {
			group["url"] = "https://www.gstatic.com/generate_204"
		}
		if (groupType == "url-test" || groupType == "fallback" || groupType == "load-balance") && intFromAny(group["interval"]) == 0 {
			group["interval"] = 300
		}
	}
	return normalized
}

func firstClashGroupName(groups []map[string]any, fallback string) string {
	for _, group := range groups {
		if name := strings.TrimSpace(fmt.Sprint(group["name"])); name != "" {
			return name
		}
	}
	return fallback
}

func (h Handler) generateV2RaySubscription() string {
	proxies := h.clashProxies()
	lines := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		if share := proxyShareURI(proxy); share != "" {
			lines = append(lines, share)
		}
	}
	return strings.Join(lines, "\n")
}

func (h Handler) generateSingBoxConfig() map[string]any {
	proxies := h.clashProxies()
	outbounds := []map[string]any{
		{"type": "selector", "tag": "OU-Auto", "outbounds": proxyNamesForSingBox(proxies)},
		{"type": "urltest", "tag": "OU-URLTest", "outbounds": proxyNamesForSingBox(proxies), "url": "https://www.gstatic.com/generate_204", "interval": "5m"},
	}
	for _, proxy := range proxies {
		if outbound := singBoxOutboundFromProxy(proxy); len(outbound) > 0 {
			outbounds = append(outbounds, outbound)
		}
	}
	outbounds = append(outbounds, map[string]any{"type": "direct", "tag": "DIRECT"})
	return map[string]any{
		"log": gin.H{"level": "warn"},
		"inbounds": []map[string]any{
			{"type": "mixed", "tag": "mixed-in", "listen": "127.0.0.1", "listen_port": 7890},
		},
		"outbounds": outbounds,
		"route":     gin.H{"final": "OU-Auto"},
	}
}

func proxyNamesForSingBox(proxies []map[string]any) []string {
	names := make([]string, 0, len(proxies)+1)
	for _, proxy := range proxies {
		if name := proxyString(proxy, "name"); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		names = append(names, "DIRECT")
	}
	return names
}

func singBoxOutboundFromProxy(proxy map[string]any) map[string]any {
	name := proxyString(proxy, "name")
	server := proxyString(proxy, "server")
	port := proxyPort(proxy)
	if name == "" || server == "" || port <= 0 {
		return nil
	}
	outbound := map[string]any{
		"tag":         name,
		"server":      server,
		"server_port": port,
	}
	switch clashType(proxyString(proxy, "type")) {
	case "ss":
		outbound["type"] = "shadowsocks"
		outbound["method"] = proxyString(proxy, "cipher", "method")
		outbound["password"] = proxyString(proxy, "password")
	case "vmess":
		outbound["type"] = "vmess"
		outbound["uuid"] = proxyString(proxy, "uuid", "id")
		outbound["security"] = defaultNonEmpty(proxyString(proxy, "security", "cipher"), "auto")
		if alterID := intFromAny(firstProxyValue(proxy, "alterId", "alter-id", "alter_id", "aid")); alterID > 0 {
			outbound["alter_id"] = alterID
		}
	case "vless":
		outbound["type"] = "vless"
		outbound["uuid"] = proxyString(proxy, "uuid", "id")
		if flow := proxyString(proxy, "flow"); flow != "" {
			outbound["flow"] = flow
		}
		addSingBoxTLS(outbound, proxy)
	case "trojan":
		outbound["type"] = "trojan"
		outbound["password"] = proxyString(proxy, "password")
		addSingBoxTLS(outbound, proxy)
	case "hysteria2":
		outbound["type"] = "hysteria2"
		outbound["password"] = proxyString(proxy, "password", "auth")
		addSingBoxTLS(outbound, proxy)
	default:
		return nil
	}
	return outbound
}

func addSingBoxTLS(outbound, proxy map[string]any) {
	security := strings.ToLower(proxyString(proxy, "security"))
	tlsEnabled := security == "tls" || security == "reality" || boolValue(firstProxyValue(proxy, "tls"))
	if !tlsEnabled {
		return
	}
	tlsConfig := map[string]any{"enabled": true}
	if serverName := proxyString(proxy, "sni", "servername", "server_name"); serverName != "" {
		tlsConfig["server_name"] = serverName
	}
	if boolValue(firstProxyValue(proxy, "skip-cert-verify", "skip_cert_verify", "allowInsecure")) {
		tlsConfig["insecure"] = true
	}
	if security == "reality" || proxyString(proxy, "pbk", "public-key", "public_key") != "" {
		reality := map[string]any{"enabled": true}
		if publicKey := proxyString(proxy, "pbk", "public-key", "public_key"); publicKey != "" {
			reality["public_key"] = publicKey
		}
		if shortID := proxyString(proxy, "sid", "short-id", "short_id"); shortID != "" {
			reality["short_id"] = shortID
		}
		tlsConfig["reality"] = reality
	}
	outbound["tls"] = tlsConfig
}

func proxyShareURI(proxy map[string]any) string {
	name := proxyString(proxy, "name")
	server := proxyString(proxy, "server")
	port := proxyPort(proxy)
	if name == "" || server == "" || port <= 0 {
		return ""
	}
	fragment := url.QueryEscape(name)
	switch clashType(proxyString(proxy, "type")) {
	case "ss":
		method := proxyString(proxy, "cipher", "method")
		password := proxyString(proxy, "password")
		if method == "" || password == "" {
			return ""
		}
		user := base64.RawURLEncoding.EncodeToString([]byte(method + ":" + password))
		return fmt.Sprintf("ss://%s@%s#%s", user, net.JoinHostPort(server, strconv.Itoa(port)), fragment)
	case "trojan":
		password := proxyString(proxy, "password")
		if password == "" {
			return ""
		}
		return shareWithUserInfo("trojan", password, server, port, shareQuery(proxy, "sni", "alpn", "network", "ws-opts", "grpc-opts", "skip-cert-verify"), fragment)
	case "vless":
		uuid := proxyString(proxy, "uuid", "id")
		if uuid == "" {
			return ""
		}
		query := shareQuery(proxy, "encryption", "flow", "security", "sni", "fp", "pbk", "sid", "type", "host", "path", "serviceName")
		if query.Get("encryption") == "" {
			query.Set("encryption", "none")
		}
		return shareWithUserInfo("vless", uuid, server, port, query, fragment)
	case "vmess":
		body := map[string]any{
			"v":    "2",
			"ps":   name,
			"add":  server,
			"port": strconv.Itoa(port),
			"id":   proxyString(proxy, "uuid", "id"),
			"aid":  strconv.Itoa(intFromAny(firstProxyValue(proxy, "alterId", "alter-id", "alter_id", "aid"))),
			"net":  defaultNonEmpty(proxyString(proxy, "network", "type"), "tcp"),
			"type": "none",
			"host": proxyString(proxy, "host"),
			"path": proxyString(proxy, "path"),
			"tls":  proxyString(proxy, "tls", "security"),
			"sni":  proxyString(proxy, "sni", "servername"),
		}
		content, _ := json.Marshal(body)
		return "vmess://" + base64.StdEncoding.EncodeToString(content)
	case "hysteria2":
		password := proxyString(proxy, "password", "auth")
		if password == "" {
			return ""
		}
		return shareWithUserInfo("hysteria2", password, server, port, shareQuery(proxy, "sni", "obfs", "obfs-password", "insecure"), fragment)
	default:
		return ""
	}
}

func shareWithUserInfo(scheme, userInfo, server string, port int, query url.Values, fragment string) string {
	raw := fmt.Sprintf("%s://%s@%s", scheme, url.QueryEscape(userInfo), net.JoinHostPort(server, strconv.Itoa(port)))
	if encoded := query.Encode(); encoded != "" {
		raw += "?" + encoded
	}
	if fragment != "" {
		raw += "#" + fragment
	}
	return raw
}

func shareQuery(proxy map[string]any, keys ...string) url.Values {
	query := url.Values{}
	for _, key := range keys {
		if value := proxyString(proxy, key); value != "" {
			query.Set(key, value)
		}
	}
	return query
}

func (h Handler) listTenants(c *gin.Context) {
	var tenants []models.Tenant
	query := h.db.Order("updated_at desc")
	if !isOwner(c) {
		tenantID := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "tenantID")))
		if tenantID == "" {
			c.JSON(http.StatusOK, gin.H{"items": []models.Tenant{}})
			return
		}
		query = query.Where("id = ?", tenantID)
	}
	if err := query.Find(&tenants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query tenants failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tenants})
}

type tenantRequest struct {
	Name                string   `json:"name"`
	Status              string   `json:"status"`
	Role                string   `json:"role"`
	NodeAccess          []string `json:"nodeAccess"`
	MonthlyTrafficQuota uint64   `json:"monthlyTrafficQuota"`
	PerNodeTrafficQuota uint64   `json:"perNodeTrafficQuota"`
	MaxConnections      int      `json:"maxConnections"`
}

type tenantPatchRequest struct {
	Name                *string  `json:"name"`
	Status              *string  `json:"status"`
	Role                *string  `json:"role"`
	NodeAccess          []string `json:"nodeAccess"`
	MonthlyTrafficQuota *uint64  `json:"monthlyTrafficQuota"`
	PerNodeTrafficQuota *uint64  `json:"perNodeTrafficQuota"`
	MaxConnections      *int     `json:"maxConnections"`
}

func (h Handler) createTenant(c *gin.Context) {
	var req tenantRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	tenant := models.Tenant{
		ID:                  "ten_" + randomHex(8),
		Name:                strings.TrimSpace(req.Name),
		Status:              defaultNonEmpty(req.Status, "active"),
		Role:                defaultNonEmpty(req.Role, "operator"),
		NodeAccess:          mustJSON(req.NodeAccess),
		MonthlyTrafficQuota: req.MonthlyTrafficQuota,
		PerNodeTrafficQuota: req.PerNodeTrafficQuota,
		MaxConnections:      req.MaxConnections,
	}
	if err := h.db.Create(&tenant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create tenant failed"})
		return
	}
	h.audit("panel", "tenant.create", tenant.ID, tenant.Name)
	c.JSON(http.StatusOK, tenant)
}

func (h Handler) updateTenant(c *gin.Context) {
	var tenant models.Tenant
	if err := h.db.First(&tenant, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	var req tenantPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		tenant.Name = strings.TrimSpace(*req.Name)
	}
	if req.Status != nil {
		tenant.Status = normalizeRBACStatus(*req.Status)
	}
	if req.Role != nil {
		tenant.Role = normalizeRBACRole(*req.Role)
	}
	if req.NodeAccess != nil {
		tenant.NodeAccess = mustJSON(compactStringList(req.NodeAccess))
	}
	if req.MonthlyTrafficQuota != nil {
		tenant.MonthlyTrafficQuota = *req.MonthlyTrafficQuota
	}
	if req.PerNodeTrafficQuota != nil {
		tenant.PerNodeTrafficQuota = *req.PerNodeTrafficQuota
	}
	if req.MaxConnections != nil {
		tenant.MaxConnections = *req.MaxConnections
	}
	if err := h.db.Save(&tenant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update tenant failed"})
		return
	}
	h.audit("panel", "tenant.update", tenant.ID, tenant.Name)
	c.JSON(http.StatusOK, tenant)
}

func (h Handler) listPanelUsers(c *gin.Context) {
	var users []models.PanelUser
	query := h.db.Order("updated_at desc")
	if !isOwner(c) {
		actor := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "actor")))
		query = query.Where("id = ?", actor)
	}
	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query users failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": users})
}

type panelUserRequest struct {
	TenantID            string   `json:"tenantId"`
	Username            string   `json:"username"`
	Password            string   `json:"password"`
	Role                string   `json:"role"`
	Status              string   `json:"status"`
	NodeAccess          []string `json:"nodeAccess"`
	MonthlyTrafficQuota uint64   `json:"monthlyTrafficQuota"`
	PerNodeTrafficQuota uint64   `json:"perNodeTrafficQuota"`
	MaxConnections      int      `json:"maxConnections"`
}

type panelUserPatchRequest struct {
	TenantID            *string  `json:"tenantId"`
	Username            *string  `json:"username"`
	Password            *string  `json:"password"`
	Role                *string  `json:"role"`
	Status              *string  `json:"status"`
	NodeAccess          []string `json:"nodeAccess"`
	MonthlyTrafficQuota *uint64  `json:"monthlyTrafficQuota"`
	PerNodeTrafficQuota *uint64  `json:"perNodeTrafficQuota"`
	MaxConnections      *int     `json:"maxConnections"`
}

func (h Handler) createPanelUser(c *gin.Context) {
	var req panelUserRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Username) == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID != "" {
		var tenant models.Tenant
		if err := h.db.Select("id").Where("id = ? AND status = ?", tenantID, "active").First(&tenant).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant is not active"})
			return
		}
	}
	user := models.PanelUser{
		ID:                  "usr_" + randomHex(8),
		TenantID:            tenantID,
		Username:            strings.TrimSpace(req.Username),
		PasswordSHA:         hashSecret(req.Password),
		Role:                defaultNonEmpty(req.Role, "operator"),
		Status:              defaultNonEmpty(req.Status, "active"),
		NodeAccess:          mustJSON(req.NodeAccess),
		MonthlyTrafficQuota: req.MonthlyTrafficQuota,
		PerNodeTrafficQuota: req.PerNodeTrafficQuota,
		MaxConnections:      req.MaxConnections,
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create user failed"})
		return
	}
	h.audit("panel", "user.create", user.ID, user.Username)
	c.JSON(http.StatusOK, user)
}

func (h Handler) updatePanelUser(c *gin.Context) {
	var user models.PanelUser
	if err := h.db.First(&user, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	var req panelUserPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.TenantID != nil {
		tenantID := strings.TrimSpace(*req.TenantID)
		if tenantID != "" {
			var tenant models.Tenant
			if err := h.db.Select("id").Where("id = ? AND status = ?", tenantID, "active").First(&tenant).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "tenant is not active"})
				return
			}
		}
		user.TenantID = tenantID
	}
	if req.Username != nil && strings.TrimSpace(*req.Username) != "" {
		user.Username = strings.TrimSpace(*req.Username)
	}
	if req.Password != nil && *req.Password != "" {
		user.PasswordSHA = hashSecret(*req.Password)
	}
	if req.Role != nil {
		user.Role = normalizeRBACRole(*req.Role)
	}
	if req.Status != nil {
		user.Status = normalizeRBACStatus(*req.Status)
	}
	if req.NodeAccess != nil {
		user.NodeAccess = mustJSON(compactStringList(req.NodeAccess))
	}
	if req.MonthlyTrafficQuota != nil {
		user.MonthlyTrafficQuota = *req.MonthlyTrafficQuota
	}
	if req.PerNodeTrafficQuota != nil {
		user.PerNodeTrafficQuota = *req.PerNodeTrafficQuota
	}
	if req.MaxConnections != nil {
		user.MaxConnections = *req.MaxConnections
	}
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update user failed"})
		return
	}
	h.audit("panel", "user.update", user.ID, user.Username)
	c.JSON(http.StatusOK, user)
}

func normalizeRBACRole(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "owner":
		return "owner"
	default:
		return "operator"
	}
}

func normalizeRBACStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "paused", "disabled", "inactive":
		return "paused"
	default:
		return "active"
	}
}

type apiKeyRequest struct {
	TenantID  string     `json:"tenantId"`
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type apiKeyPatchRequest struct {
	TenantID  *string    `json:"tenantId"`
	Name      *string    `json:"name"`
	Scopes    []string   `json:"scopes"`
	Status    *string    `json:"status"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

func (h Handler) listAPIKeys(c *gin.Context) {
	var keys []models.APIKey
	if err := h.db.Order("updated_at desc").Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query api keys failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": keys})
}

func (h Handler) createAPIKey(c *gin.Context) {
	var req apiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID != "" {
		var tenant models.Tenant
		if err := h.db.Select("id").Where("id = ? AND status = ?", tenantID, "active").First(&tenant).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant is not active"})
			return
		}
	}
	scopes := compactStringList(req.Scopes)
	if len(scopes) == 0 {
		scopes = []string{"panel:read"}
	}
	rawKey := "ouak_" + randomHex(24)
	key := models.APIKey{
		ID:        "key_" + randomHex(8),
		TenantID:  tenantID,
		Name:      strings.TrimSpace(req.Name),
		KeyHash:   hashSecret(rawKey),
		Scopes:    mustJSON(scopes),
		Status:    normalizeAPIKeyStatus(req.Status),
		ExpiresAt: req.ExpiresAt,
	}
	if err := h.db.Create(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create api key failed"})
		return
	}
	h.audit("panel", "apikey.create", key.ID, key.Name)
	c.JSON(http.StatusOK, gin.H{"item": key, "apiKey": rawKey})
}

func (h Handler) updateAPIKey(c *gin.Context) {
	var key models.APIKey
	if err := h.db.First(&key, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "api key not found"})
		return
	}
	var req apiKeyPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.TenantID != nil {
		tenantID := strings.TrimSpace(*req.TenantID)
		if tenantID != "" {
			var tenant models.Tenant
			if err := h.db.Select("id").Where("id = ? AND status = ?", tenantID, "active").First(&tenant).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "tenant is not active"})
				return
			}
		}
		key.TenantID = tenantID
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		key.Name = strings.TrimSpace(*req.Name)
	}
	if req.Scopes != nil {
		scopes := compactStringList(req.Scopes)
		if len(scopes) == 0 {
			scopes = []string{"panel:read"}
		}
		key.Scopes = mustJSON(scopes)
	}
	if req.Status != nil {
		key.Status = normalizeAPIKeyStatus(*req.Status)
	}
	if req.ExpiresAt != nil {
		key.ExpiresAt = req.ExpiresAt
	}
	if err := h.db.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update api key failed"})
		return
	}
	h.audit("panel", "apikey.update", key.ID, key.Name)
	c.JSON(http.StatusOK, key)
}

func (h Handler) revokeAPIKey(c *gin.Context) {
	var key models.APIKey
	if err := h.db.First(&key, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "api key not found"})
		return
	}
	key.Status = "revoked"
	if err := h.db.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "revoke api key failed"})
		return
	}
	h.audit("panel", "apikey.revoke", key.ID, key.Name)
	c.JSON(http.StatusOK, gin.H{"ok": true, "item": key})
}

func normalizeAPIKeyStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "paused", "disabled", "inactive":
		return "paused"
	case "revoked":
		return "revoked"
	default:
		return "active"
	}
}

func (h Handler) apiDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"openapi": "3.1.0",
		"info": gin.H{
			"title":       "OU-UI REST API",
			"version":     "v6.0.0",
			"description": "Authenticated control-plane API for OU-UI agents, generated proxy nodes, routing, HA, alerts, subscriptions, RBAC, API keys, and Copilot operations.",
		},
		"servers":  []gin.H{{"url": h.cfg.SecurePath + "/api/v1", "description": "Current panel secure-path API root"}},
		"security": []gin.H{{"bearerAuth": []string{}}},
		"tags": []gin.H{
			{"name": "Auth"}, {"name": "Agents"}, {"name": "Nodes"}, {"name": "Traffic"},
			{"name": "Routing"}, {"name": "High Availability"}, {"name": "Alerts"},
			{"name": "Subscriptions"}, {"name": "Clash"}, {"name": "RBAC"}, {"name": "Integrations"}, {"name": "Copilot"},
		},
		"components": gin.H{
			"securitySchemes": gin.H{
				"bearerAuth": gin.H{"type": "http", "scheme": "bearer", "bearerFormat": "JWT or OU-UI API key"},
			},
			"schemas": gin.H{
				"ListResponse": gin.H{"type": "object", "properties": gin.H{"items": gin.H{"type": "array", "items": gin.H{"type": "object"}}}},
				"Error":        gin.H{"type": "object", "properties": gin.H{"error": gin.H{"type": "string"}}},
			},
		},
		"paths": openAPIPaths(),
		"x-ou-ui-alert-events": []string{
			"cpu.high",
			"cpu.overload",
			"agent.traffic.overload",
			"traffic.quota.warning",
			"traffic.quota.exceeded",
			"node.traffic.overload",
			"node.connections.high",
			"agent.offline",
			"agent.error",
			"tenant.quota.exceeded",
			"tenant.node.quota.exceeded",
			"user.quota.exceeded",
			"user.node.quota.exceeded",
		},
		"x-ou-ui-scopes": []string{"panel:read", "panel:write", "panel:*", "*"},
	})
}

func openAPIPaths() gin.H {
	return gin.H{
		"/auth/login":                       gin.H{"post": apiOperation("Auth", "Create a panel session token", "panel:read", gin.H{"username": "admin", "password": "secret"})},
		"/overview":                         gin.H{"get": apiOperation("Agents", "Read tenant-filtered control-plane overview", "panel:read", nil)},
		"/agents":                           gin.H{"get": apiOperation("Agents", "List visible Agents", "panel:read", nil)},
		"/agents/install-script":            gin.H{"get": apiOperation("Agents", "Render the Agent install script", "panel:read", nil)},
		"/agents/{id}/network-optimization": gin.H{"post": apiOperation("Agents", "Queue BBR/sysctl host optimization", "panel:write", gin.H{"profile": "bbr-v3", "rebootPolicy": "manual"}, pathParam("id", "Agent ID"))},
		"/nodes":                            gin.H{"get": apiOperation("Nodes", "List visible generated proxy nodes", "panel:read", nil), "post": apiOperation("Nodes", "Create a managed proxy node and deployment task", "panel:write", gin.H{"agentId": "agt_x", "runtime": "xray", "protocol": "vless", "port": 443})},
		"/traffic/nodes":                    gin.H{"get": apiOperation("Traffic", "List latest per-node traffic samples", "panel:read", nil)},
		"/traffic/nodes/{id}/samples":       gin.H{"get": apiOperation("Traffic", "List historical samples for one node", "panel:read", nil, pathParam("id", "Node ID"), queryParam("limit", "Maximum samples"))},
		"/routing/rules":                    gin.H{"get": apiOperation("Routing", "List routing rules", "panel:read", nil), "post": apiOperation("Routing", "Create GeoIP/GeoSite/domain/protocol routing rule", "panel:write", gin.H{"name": "Block ads", "ruleType": "ads", "match": "category-ads-all", "action": "block"})},
		"/routing/rules/{id}":               gin.H{"patch": apiOperation("Routing", "Update a routing rule", "panel:write", gin.H{"enabled": true}, pathParam("id", "Routing rule ID")), "delete": apiOperation("Routing", "Delete a routing rule", "panel:write", nil, pathParam("id", "Routing rule ID"))},
		"/routing/apply":                    gin.H{"post": apiOperation("Routing", "Queue routing.apply tasks for capable Agents", "panel:write", gin.H{"agentIds": []string{"agt_x"}})},
		"/routing/export/xray":              gin.H{"get": apiOperation("Routing", "Export Xray routing config", "panel:read", nil)},
		"/load-balancers":                   gin.H{"get": apiOperation("High Availability", "List HA groups", "panel:read", nil), "post": apiOperation("High Availability", "Create HA group", "panel:write", gin.H{"name": "Global HA", "entryTag": "ou-ha", "strategy": "latency-loss"})},
		"/load-balancers/{id}/entry":        gin.H{"get": apiOperation("High Availability", "Resolve current HA entry decision", "panel:read", nil, pathParam("id", "Load balancer group ID"))},
		"/load-balancers/{id}/health":       gin.H{"post": apiOperation("High Availability", "Update HA member health metrics", "panel:write", gin.H{"members": []gin.H{}}, pathParam("id", "Load balancer group ID"))},
		"/webhooks":                         gin.H{"get": apiOperation("Alerts", "List alert webhooks", "panel:read", nil), "post": apiOperation("Alerts", "Create Telegram, ServerChan, or generic webhook", "panel:write", gin.H{"name": "Ops", "kind": "telegram", "eventTypes": []string{"agent.offline"}})},
		"/webhooks/{id}":                    gin.H{"patch": apiOperation("Alerts", "Update alert webhook", "panel:write", gin.H{"enabled": true}, pathParam("id", "Webhook ID"))},
		"/webhooks/{id}/test":               gin.H{"post": apiOperation("Alerts", "Send webhook test alert", "panel:write", nil, pathParam("id", "Webhook ID"))},
		"/alerts":                           gin.H{"get": apiOperation("Alerts", "List alert events and delivery status", "panel:read", nil)},
		"/subscriptions":                    gin.H{"get": apiOperation("Subscriptions", "List external subscriptions", "panel:read", nil), "post": apiOperation("Subscriptions", "Create and optionally import external subscription", "panel:write", gin.H{"name": "External pool", "url": "https://example.com/sub", "format": "auto"})},
		"/subscriptions/{id}/import":        gin.H{"post": apiOperation("Subscriptions", "Import one external subscription", "panel:write", gin.H{"content": "vmess://..."}, pathParam("id", "Subscription ID"))},
		"/subscriptions/aggregate":          gin.H{"get": apiOperation("Subscriptions", "Generate aggregated Clash, V2Ray, raw share, or Sing-box subscription", "panel:read", nil, queryParam("format", "clash, v2ray, raw, or sing-box"))},
		"/external-nodes":                   gin.H{"get": apiOperation("Subscriptions", "List imported external nodes", "panel:read", nil, queryParam("subscriptionId", "Optional subscription filter"))},
		"/clash/profiles":                   gin.H{"get": apiOperation("Clash", "List Clash profiles", "panel:read", nil), "post": apiOperation("Clash", "Create hosted Clash YAML profile", "panel:write", gin.H{"name": "OU-UI Clash", "ruleProviders": []gin.H{}, "proxyGroups": []gin.H{}, "selectedNodes": []string{"*"}})},
		"/clash/profiles/{id}.yaml":         gin.H{"get": apiOperation("Clash", "Download hosted Clash profile YAML", "panel:read", nil, pathParam("id", "Clash profile ID"))},
		"/tenants":                          gin.H{"get": apiOperation("RBAC", "List tenants", "panel:read", nil), "post": apiOperation("RBAC", "Create tenant with node and quota limits", "panel:write", gin.H{"name": "Ops", "nodeAccess": []string{"agt_x"}, "monthlyTrafficQuota": 1073741824, "perNodeTrafficQuota": 268435456, "maxConnections": 1000})},
		"/tenants/{id}":                     gin.H{"patch": apiOperation("RBAC", "Update tenant status, node access, and quota limits", "panel:write", gin.H{"status": "paused", "nodeAccess": []string{"agt_x"}, "monthlyTrafficQuota": 1073741824}, pathParam("id", "Tenant ID"))},
		"/users":                            gin.H{"get": apiOperation("RBAC", "List sub-users", "panel:read", nil), "post": apiOperation("RBAC", "Create panel sub-user", "panel:write", gin.H{"username": "operator", "password": "change-me", "tenantId": "ten_x"})},
		"/users/{id}":                       gin.H{"patch": apiOperation("RBAC", "Update sub-user tenant binding, status, password, node access, and quotas", "panel:write", gin.H{"status": "paused", "nodeAccess": []string{"agt_x"}, "maxConnections": 1000}, pathParam("id", "Panel user ID"))},
		"/api-keys":                         gin.H{"get": apiOperation("Integrations", "List API key metadata for integration governance", "panel:read", nil), "post": apiOperation("Integrations", "Create scoped API key for third-party systems", "panel:write", gin.H{"name": "Billing", "scopes": []string{"panel:read"}, "expiresAt": "2026-12-31T00:00:00Z"})},
		"/api-keys/{id}":                    gin.H{"patch": apiOperation("Integrations", "Update API key tenant, scopes, status, or expiry", "panel:write", gin.H{"status": "paused", "scopes": []string{"panel:read"}}, pathParam("id", "API key ID")), "delete": apiOperation("Integrations", "Revoke an API key without removing audit metadata", "panel:write", nil, pathParam("id", "API key ID"))},
		"/api-docs":                         gin.H{"get": apiOperation("Integrations", "Read OpenAPI document", "panel:read", nil)},
		"/copilot/incidents":                gin.H{"get": apiOperation("Copilot", "List Copilot incidents", "panel:read", nil)},
		"/copilot/ask":                      gin.H{"post": apiOperation("Copilot", "Ask AI operations Copilot", "panel:write", gin.H{"question": "Why is my Agent degraded?"})},
		"/tasks":                            gin.H{"get": apiOperation("Agents", "List visible tasks", "panel:read", nil), "post": apiOperation("Agents", "Create a direct Agent task", "panel:write", gin.H{"agentId": "agt_x", "type": "noop"})},
	}
}

func apiOperation(tag, summary, scope string, requestExample any, parameters ...gin.H) gin.H {
	op := gin.H{
		"tags":        []string{tag},
		"summary":     summary,
		"security":    []gin.H{{"bearerAuth": []string{scope}}},
		"responses":   apiResponses(),
		"x-ou-scope":  scope,
		"operationId": operationID(summary),
	}
	if requestExample != nil {
		op["requestBody"] = gin.H{
			"required": true,
			"content":  gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}, "example": requestExample}},
		}
	}
	if len(parameters) > 0 {
		op["parameters"] = parameters
	}
	return op
}

func apiResponses() gin.H {
	return gin.H{
		"200": gin.H{"description": "Successful response"},
		"400": gin.H{"description": "Invalid request"},
		"401": gin.H{"description": "Missing or invalid bearer token"},
		"403": gin.H{"description": "Scope, tenant, node access, or quota denied"},
	}
}

func pathParam(name, description string) gin.H {
	return gin.H{"name": name, "in": "path", "required": true, "description": description, "schema": gin.H{"type": "string"}}
}

func queryParam(name, description string) gin.H {
	return gin.H{"name": name, "in": "query", "required": false, "description": description, "schema": gin.H{"type": "string"}}
}

func operationID(summary string) string {
	words := strings.FieldsFunc(summary, func(r rune) bool {
		return r == ' ' || r == '-' || r == '/' || r == ','
	})
	for i, word := range words {
		word = strings.TrimFunc(word, func(r rune) bool {
			return r < '0' || (r > '9' && r < 'A') || (r > 'Z' && r < 'a') || r > 'z'
		})
		if word == "" {
			continue
		}
		if i == 0 {
			words[i] = strings.ToLower(word[:1]) + word[1:]
		} else {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, "")
}

type copilotRequest struct {
	Question string         `json:"question"`
	Context  map[string]any `json:"context"`
}

func (h Handler) listCopilotIncidents(c *gin.Context) {
	var incidents []models.CopilotIncident
	query := h.db.Order("created_at desc").Limit(limitFromQuery(c, 50))
	if !isOwner(c) {
		actor := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "actor")))
		query = query.Where("context LIKE ?", "%\"actor\":\""+actor+"\"%")
	}
	if err := query.Find(&incidents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query copilot incidents failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": incidents})
}

func (h Handler) askCopilot(c *gin.Context) {
	var req copilotRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Question) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	context := h.collectCopilotContext(c, req.Context)
	answer := h.localCopilotAnswer(req.Question, context)
	status := "local"
	model := "ou-ui-rulebook"
	if remote, remoteModel, err := h.remoteCopilotAnswer(req.Question, context); err == nil && strings.TrimSpace(remote) != "" {
		answer = remote
		status = "remote"
		model = remoteModel
	}
	rawContext, _ := json.Marshal(context)
	incident := models.CopilotIncident{
		ID:       "cop_" + randomHex(8),
		Question: strings.TrimSpace(req.Question),
		Context:  datatypes.JSON(rawContext),
		Answer:   answer,
		Model:    model,
		Status:   status,
	}
	if err := h.db.Create(&incident).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "record copilot incident failed"})
		return
	}
	c.JSON(http.StatusOK, incident)
}

func (h Handler) collectCopilotContext(c *gin.Context, extra map[string]any) map[string]any {
	actor := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "actor")))
	tenantID := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "tenantID")))
	role := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "role")))
	allowed, limited := nodeAccessFilter(c)
	var alerts []models.AlertEvent
	alertQuery := h.db.Order("created_at desc").Limit(10)
	if limited {
		alertQuery = alertQuery.Where(
			"(source_type IN ? AND source_id IN ?) OR (source_type = ? AND source_id = ?) OR (source_type = ? AND source_id = ?)",
			[]string{"agent", "node"}, allowed,
			"tenant", tenantID,
			"user", actor,
		)
	}
	_ = alertQuery.Find(&alerts).Error
	var tasks []models.Task
	taskQuery := h.db.Order("created_at desc").Limit(10)
	if limited {
		taskQuery = taskQuery.Where("agent_id IN ?", allowed)
	}
	_ = taskQuery.Find(&tasks).Error
	for i := range tasks {
		tasks[i].Payload = redactJSON(tasks[i].Payload)
		tasks[i].Result = redactJSON(tasks[i].Result)
	}
	var agents []models.Agent
	agentQuery := h.db.Order("updated_at desc").Limit(20)
	if limited {
		agentQuery = agentQuery.Where("id IN ?", allowed)
	}
	_ = agentQuery.Find(&agents).Error
	for i := range agents {
		h.decorateAgent(&agents[i])
		agents[i].AgentTokenSHA = ""
	}
	return map[string]any{
		"actor":      actor,
		"tenantId":   tenantID,
		"role":       role,
		"nodeAccess": allowed,
		"extra":      extra,
		"alerts":     alerts,
		"tasks":      tasks,
		"agents":     agents,
	}
}

func (h Handler) localCopilotAnswer(question string, context map[string]any) string {
	var b strings.Builder
	b.WriteString("OU-UI Copilot local diagnosis\n")
	b.WriteString("Question: ")
	b.WriteString(strings.TrimSpace(question))
	b.WriteString("\n\nRecommended actions:\n")
	b.WriteString("1. Check the newest alert and failed task first; most OU-UI incidents are caused by runtime deploy health gates, host overload, or expired Agent heartbeats.\n")
	b.WriteString("2. For node deploy failures, inspect the task result stages and run the rendered runtime config through the native binary validation command before restarting service.\n")
	b.WriteString("3. For traffic anomalies, compare /traffic/nodes latest samples with the Agent aggregate heartbeat to isolate one generated node from host-level noise.\n")
	b.WriteString("4. For routing issues, export /routing/export/xray and verify GeoIP/GeoSite tags map to direct, blocked, or proxy outbound tags.\n")
	b.WriteString("5. For host tuning regressions, rerun host.optimize with profile conservative and keep rebootPolicy manual until BBR is verified.\n")
	return b.String()
}

func (h Handler) remoteCopilotAnswer(question string, context map[string]any) (string, string, error) {
	if h.cfg.LLMEndpoint == "" || h.cfg.LLMAPIKey == "" {
		return "", "", fmt.Errorf("llm endpoint is not configured")
	}
	model := h.cfg.LLMModel
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are OU-UI's SRE Copilot. Return concise troubleshooting steps and concrete shell commands when safe."},
			{"role": "user", "content": fmt.Sprintf("Question:\n%s\n\nContext JSON:\n%s", question, compactJSON(context))},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(h.cfg.LLMEndpoint, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", model, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.cfg.LLMAPIKey)
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return "", model, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", model, fmt.Errorf("llm returned %s", resp.Status)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", model, err
	}
	if len(out.Choices) == 0 {
		return "", model, fmt.Errorf("llm returned no choices")
	}
	return out.Choices[0].Message.Content, model, nil
}

func parseExternalNodes(subscriptionID, content string) []models.ExternalNode {
	content = decodeMaybeBase64(strings.TrimSpace(content))
	nodes := parseClashYAMLNodes(subscriptionID, content)
	if len(nodes) > 0 {
		return nodes
	}
	nodes = parseSingBoxJSONNodes(subscriptionID, content)
	if len(nodes) > 0 {
		return nodes
	}
	lines := strings.Split(content, "\n")
	for index, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- {") || strings.HasPrefix(line, "{") {
			if node, ok := parseClashInlineProxy(subscriptionID, line, index); ok {
				nodes = append(nodes, node)
			}
			continue
		}
		if !strings.Contains(line, "://") {
			continue
		}
		if node, ok := parseShareURI(subscriptionID, line, index); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func parseClashYAMLNodes(subscriptionID, content string) []models.ExternalNode {
	var doc struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil || len(doc.Proxies) == 0 {
		return nil
	}
	nodes := make([]models.ExternalNode, 0, len(doc.Proxies))
	for index, proxy := range doc.Proxies {
		node, ok := externalNodeFromProxyMap(subscriptionID, proxy, index)
		if ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func parseSingBoxJSONNodes(subscriptionID, content string) []models.ExternalNode {
	var doc struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&doc); err != nil || len(doc.Outbounds) == 0 {
		return nil
	}
	nodes := make([]models.ExternalNode, 0, len(doc.Outbounds))
	for index, outbound := range doc.Outbounds {
		node, ok := externalNodeFromSingBoxOutbound(subscriptionID, outbound, index)
		if ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func externalNodeFromSingBoxOutbound(subscriptionID string, outbound map[string]any, index int) (models.ExternalNode, bool) {
	protocol := strings.ToLower(strings.TrimSpace(fmt.Sprint(outbound["type"])))
	if !isProxyProtocol(protocol) {
		return models.ExternalNode{}, false
	}
	host := strings.TrimSpace(fmt.Sprint(outbound["server"]))
	port := intFromAny(firstProxyValue(outbound, "server_port", "serverPort", "port"))
	name := strings.TrimSpace(fmt.Sprint(outbound["tag"]))
	if name == "" {
		name = defaultNonEmpty(protocol, "proxy") + "-" + strconv.Itoa(index+1)
	}
	raw, _ := json.Marshal(outbound)
	return models.ExternalNode{
		ID:             stableExternalNodeID(subscriptionID, string(raw)),
		SubscriptionID: subscriptionID,
		Name:           name,
		Protocol:       protocol,
		Address:        host,
		Port:           port,
		Source:         "sing-box",
		Config:         mustJSON(outbound),
		Enabled:        true,
	}, host != "" && port > 0
}

func externalNodeFromProxyMap(subscriptionID string, config map[string]any, index int) (models.ExternalNode, bool) {
	name := strings.TrimSpace(fmt.Sprint(config["name"]))
	protocol := strings.TrimSpace(fmt.Sprint(config["type"]))
	host := strings.TrimSpace(fmt.Sprint(config["server"]))
	port := intFromAny(config["port"])
	if name == "" {
		name = defaultNonEmpty(protocol, "proxy") + "-" + strconv.Itoa(index+1)
	}
	raw, _ := json.Marshal(config)
	return models.ExternalNode{
		ID:             stableExternalNodeID(subscriptionID, string(raw)),
		SubscriptionID: subscriptionID,
		Name:           name,
		Protocol:       protocol,
		Address:        host,
		Port:           port,
		Source:         "clash",
		Config:         mustJSON(config),
		Enabled:        true,
	}, protocol != "" && host != "" && port > 0
}

func parseShareURI(subscriptionID, raw string, index int) (models.ExternalNode, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return models.ExternalNode{}, false
	}
	protocol := strings.ToLower(parsed.Scheme)
	if protocol == "vmess" {
		return parseVMess(subscriptionID, raw, index)
	}
	if protocol == "ss" || protocol == "shadowsocks" {
		return parseShadowsocks(subscriptionID, raw, index)
	}
	name, _ := url.QueryUnescape(parsed.Fragment)
	if name == "" {
		name = protocol + "-" + strconv.Itoa(index+1)
	}
	host := parsed.Hostname()
	port, _ := strconv.Atoi(parsed.Port())
	config := map[string]any{"raw": raw}
	if parsed.User != nil {
		username, _ := url.QueryUnescape(parsed.User.Username())
		password, hasPassword := parsed.User.Password()
		password, _ = url.QueryUnescape(password)
		switch protocol {
		case "trojan", "hysteria2", "hy2":
			if username != "" {
				config["password"] = username
			}
			if hasPassword && password != "" {
				config["password"] = password
			}
		default:
			if username != "" {
				config["uuid"] = username
			}
			if hasPassword && password != "" {
				config["password"] = password
			}
		}
	}
	for key, values := range parsed.Query() {
		if len(values) > 0 {
			config[key] = values[0]
		}
	}
	return models.ExternalNode{
		ID:             stableExternalNodeID(subscriptionID, raw),
		SubscriptionID: subscriptionID,
		Name:           name,
		Protocol:       protocol,
		Address:        host,
		Port:           port,
		Source:         "subscription",
		Config:         mustJSON(config),
		Enabled:        true,
	}, host != "" && port > 0
}

func parseShadowsocks(subscriptionID, raw string, index int) (models.ExternalNode, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return models.ExternalNode{}, false
	}
	name, _ := url.QueryUnescape(parsed.Fragment)
	if name == "" {
		name = "ss-" + strconv.Itoa(index+1)
	}
	config := map[string]any{"raw": raw}
	for key, values := range parsed.Query() {
		if len(values) > 0 {
			config[key] = values[0]
		}
	}
	host := parsed.Hostname()
	port, _ := strconv.Atoi(parsed.Port())
	method := ""
	password := ""
	if parsed.User != nil {
		username, _ := url.QueryUnescape(parsed.User.Username())
		if parsedPassword, ok := parsed.User.Password(); ok {
			method = username
			password, _ = url.QueryUnescape(parsedPassword)
		} else {
			method, password = splitShadowsocksUserInfo(decodeMaybeBase64(username))
		}
	}
	if host == "" || port == 0 {
		encoded := parsed.Host
		if encoded == "" {
			encoded = strings.TrimPrefix(raw, "ss://")
			encoded = strings.SplitN(encoded, "#", 2)[0]
			encoded = strings.SplitN(encoded, "?", 2)[0]
		}
		decoded := decodeMaybeBase64(encoded)
		decoded = strings.SplitN(decoded, "#", 2)[0]
		decoded = strings.SplitN(decoded, "?", 2)[0]
		if methodFromAll, passwordFromAll, hostFromAll, portFromAll, ok := splitShadowsocksAuthority(decoded); ok {
			method = methodFromAll
			password = passwordFromAll
			host = hostFromAll
			port = portFromAll
		}
	}
	if method != "" {
		config["cipher"] = method
		config["method"] = method
	}
	if password != "" {
		config["password"] = password
	}
	return models.ExternalNode{
		ID:             stableExternalNodeID(subscriptionID, raw),
		SubscriptionID: subscriptionID,
		Name:           name,
		Protocol:       "ss",
		Address:        host,
		Port:           port,
		Source:         "subscription",
		Config:         mustJSON(config),
		Enabled:        true,
	}, host != "" && port > 0 && method != "" && password != ""
}

func parseVMess(subscriptionID, raw string, index int) (models.ExternalNode, bool) {
	encoded := strings.TrimPrefix(raw, "vmess://")
	decoded := decodeMaybeBase64(encoded)
	var body map[string]any
	if err := json.Unmarshal([]byte(decoded), &body); err != nil {
		return models.ExternalNode{}, false
	}
	body["raw"] = raw
	name := strings.TrimSpace(fmt.Sprint(body["ps"]))
	if name == "" {
		name = "vmess-" + strconv.Itoa(index+1)
	}
	host := strings.TrimSpace(fmt.Sprint(body["add"]))
	port := intFromAny(body["port"])
	return models.ExternalNode{
		ID:             stableExternalNodeID(subscriptionID, raw),
		SubscriptionID: subscriptionID,
		Name:           name,
		Protocol:       "vmess",
		Address:        host,
		Port:           port,
		Source:         "subscription",
		Config:         mustJSON(body),
		Enabled:        true,
	}, host != "" && port > 0
}

func parseClashInlineProxy(subscriptionID, line string, index int) (models.ExternalNode, bool) {
	line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
	line = strings.TrimSpace(strings.Trim(line, "{}"))
	fields := splitCSVRespectQuotes(line)
	config := map[string]any{}
	for _, field := range fields {
		parts := strings.SplitN(field, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		config[key] = value
	}
	name := strings.TrimSpace(fmt.Sprint(config["name"]))
	protocol := strings.TrimSpace(fmt.Sprint(config["type"]))
	host := strings.TrimSpace(fmt.Sprint(config["server"]))
	port := intFromAny(config["port"])
	if name == "" {
		name = defaultNonEmpty(protocol, "proxy") + "-" + strconv.Itoa(index+1)
	}
	return models.ExternalNode{
		ID:             stableExternalNodeID(subscriptionID, line),
		SubscriptionID: subscriptionID,
		Name:           name,
		Protocol:       protocol,
		Address:        host,
		Port:           port,
		Source:         "clash",
		Config:         mustJSON(config),
		Enabled:        true,
	}, protocol != "" && host != "" && port > 0
}

func isProxyProtocol(protocol string) bool {
	switch clashType(protocol) {
	case "ss", "vmess", "vless", "trojan", "hysteria2":
		return true
	default:
		return false
	}
}

func splitShadowsocksUserInfo(value string) (string, string) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func splitShadowsocksAuthority(value string) (method, password, host string, port int, ok bool) {
	userInfo, endpoint, found := strings.Cut(value, "@")
	if !found {
		return "", "", "", 0, false
	}
	method, password = splitShadowsocksUserInfo(userInfo)
	if method == "" || password == "" {
		return "", "", "", 0, false
	}
	host, portText, err := net.SplitHostPort(endpoint)
	if err != nil {
		parts := strings.Split(endpoint, ":")
		if len(parts) < 2 {
			return "", "", "", 0, false
		}
		host = strings.Join(parts[:len(parts)-1], ":")
		portText = parts[len(parts)-1]
	}
	port, err = strconv.Atoi(strings.TrimSpace(portText))
	if err != nil {
		return "", "", "", 0, false
	}
	return method, password, strings.Trim(host, "[]"), port, true
}

func fetchText(rawURL string) (string, error) {
	if err := validateSubscriptionURL(rawURL); err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "OU-UI/4.0 subscription importer")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("subscription returned %s", resp.Status)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func validateSubscriptionURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return fmt.Errorf("subscription url must use http or https")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("subscription url host is required")
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return fmt.Errorf("subscription url points to a local host")
	}
	if ip := net.ParseIP(host); ip != nil && !isPublicRoutableIP(ip) {
		return fmt.Errorf("subscription url points to a private address")
	}
	return nil
}

func isPublicRoutableIP(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified())
}

func postJSON(client *http.Client, endpoint string, payload map[string]any, secret string) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-OU-UI-Signature", webhookSignature(body, secret))
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %s", resp.Status)
	}
	return nil
}

func webhookSignature(body []byte, secret string) string {
	sum := sha256.Sum256(append([]byte(secret), body...))
	return "sha256=" + hex.EncodeToString(sum[:])
}

func normalizeRoutingRuleType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "geoip", "ip", "geosite", "domain", "protocol", "ads", "ad":
		if strings.EqualFold(value, "ad") {
			return "ads"
		}
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func normalizeRoutingAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "block", "blocked", "reject":
		return "block"
	case "direct":
		return "direct"
	case "proxy", "route":
		return "proxy"
	default:
		return ""
	}
}

func outboundTagForAction(rule models.RoutingRule) string {
	switch rule.Action {
	case "block":
		return "blocked"
	case "direct":
		return "direct"
	default:
		return defaultNonEmpty(rule.TargetTag, "proxy")
	}
}

func clashRuleLine(rule models.RoutingRule) string {
	return clashRuleLineWithTarget(rule, "OU-Auto")
}

func clashRuleLineWithTarget(rule models.RoutingRule, defaultTarget string) string {
	target := defaultNonEmpty(defaultTarget, "OU-Auto")
	if rule.Action == "block" {
		target = "REJECT"
	} else if rule.Action == "direct" {
		target = "DIRECT"
	} else if rule.TargetTag != "" {
		target = rule.TargetTag
	}
	switch rule.RuleType {
	case "geoip":
		return "GEOIP," + strings.TrimPrefix(rule.Match, "geoip:") + "," + target
	case "geosite", "ads":
		return "GEOSITE," + strings.TrimPrefix(rule.Match, "geosite:") + "," + target
	case "domain":
		return "DOMAIN-SUFFIX," + strings.TrimPrefix(rule.Match, "domain:") + "," + target
	case "ip":
		return "IP-CIDR," + rule.Match + "," + target + ",no-resolve"
	default:
		return "MATCH," + target
	}
}

func ensurePrefix(value, prefix string) string {
	value = strings.TrimSpace(value)
	if strings.Contains(value, ":") {
		return value
	}
	return prefix + value
}

func stableExternalNodeID(subscriptionID, raw string) string {
	sum := sha256.Sum256([]byte(subscriptionID + "\n" + raw))
	return "ext_" + hex.EncodeToString(sum[:])[:16]
}

func decodeMaybeBase64(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	candidates := []string{trimmed}
	if missing := len(trimmed) % 4; missing != 0 {
		candidates = append(candidates, trimmed+strings.Repeat("=", 4-missing))
	}
	for _, candidate := range candidates {
		for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
			if decoded, err := encoding.DecodeString(candidate); err == nil {
				text := strings.TrimSpace(string(decoded))
				if text != "" {
					return text
				}
			}
		}
	}
	return value
}

func splitCSVRespectQuotes(value string) []string {
	var fields []string
	var current strings.Builder
	quote := rune(0)
	for _, r := range value {
		switch {
		case quote != 0:
			current.WriteRune(r)
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
			current.WriteRune(r)
		case r == ',':
			fields = append(fields, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		fields = append(fields, strings.TrimSpace(current.String()))
	}
	return fields
}

func inlineYAML(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", key, yamlValue(values[key])))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func yamlValue(value any) string {
	switch typed := value.(type) {
	case []string:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, quoteYAML(item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, yamlValue(item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	case string:
		return quoteYAML(typed)
	case int, int64, uint64, float64, bool:
		return fmt.Sprint(typed)
	default:
		return quoteYAML(fmt.Sprint(typed))
	}
}

func quoteYAML(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	return "\"" + escaped + "\""
}

func clashType(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "shadowsocks", "ss":
		return "ss"
	case "hysteria2", "hy2":
		return "hysteria2"
	case "vless", "vmess", "trojan":
		return strings.ToLower(protocol)
	default:
		return defaultNonEmpty(protocol, "http")
	}
}

func mapFromJSON(raw datatypes.JSON) map[string]any {
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}

func mustJSON(value any) datatypes.JSON {
	content, _ := json.Marshal(value)
	return datatypes.JSON(content)
}

func compactJSON(value any) string {
	content, _ := json.Marshal(value)
	return string(content)
}

func cloneMapSlice(values []map[string]any) []map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		out = append(out, cloneMap(value))
	}
	return out
}

func cloneMap(value map[string]any) map[string]any {
	next := make(map[string]any, len(value))
	for key, item := range value {
		next[key] = item
	}
	return next
}

func containsWildcard(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "*" {
			return true
		}
	}
	return false
}

func defaultNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func limitFromQuery(c *gin.Context, fallback int) int {
	limit, err := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	if err != nil || limit <= 0 {
		return fallback
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func mapFromAny(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func boolFromAny(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		if strings.TrimSpace(typed) == "" {
			return false, false
		}
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return false, false
	}
}

func numberFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case json.Number:
		out, err := typed.Float64()
		return out, err == nil
	case string:
		if strings.TrimSpace(typed) == "" {
			return 0, false
		}
		out, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return out, err == nil
	default:
		return 0, false
	}
}

func floatFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint64:
		return float64(typed)
	case json.Number:
		out, _ := typed.Float64()
		return out
	case string:
		out, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return out
	default:
		return 0
	}
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		out, _ := typed.Int64()
		return int(out)
	case string:
		out, _ := strconv.Atoi(strings.TrimSpace(typed))
		return out
	default:
		return 0
	}
}

func firstMapString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
				return text
			}
		}
	}
	return ""
}

func firstProxyValue(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}

func proxyString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			switch typed := value.(type) {
			case string:
				if text := strings.TrimSpace(typed); text != "" {
					return text
				}
			case fmt.Stringer:
				if text := strings.TrimSpace(typed.String()); text != "" {
					return text
				}
			case bool:
				return strconv.FormatBool(typed)
			default:
				if value != nil {
					if text := strings.TrimSpace(fmt.Sprint(value)); text != "" && text != "<nil>" {
						return text
					}
				}
			}
		}
	}
	return ""
}

func proxyPort(values map[string]any) int {
	return intFromAny(firstProxyValue(values, "port", "server_port", "serverPort"))
}

func stringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return compactStringList(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, strings.TrimSpace(fmt.Sprint(item)))
		}
		return compactStringList(out)
	case string:
		return compactStringList(strings.Split(typed, ","))
	default:
		return nil
	}
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		out, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return out
	case int:
		return typed != 0
	case float64:
		return typed != 0
	default:
		return false
	}
}
