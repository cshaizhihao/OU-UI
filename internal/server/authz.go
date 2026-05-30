package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

func (h Handler) authenticateAPIKey(c *gin.Context, token string) bool {
	if !strings.HasPrefix(token, "ouak_") {
		return false
	}
	var key models.APIKey
	if err := h.db.Where("key_hash = ? AND status = ?", hashSecret(token), "active").First(&key).Error; err != nil {
		c.AbortWithStatusJSON(401, gin.H{"error": "invalid api key"})
		return true
	}
	scopes := stringListFromJSON(key.Scopes)
	nodeAccess := []string{"*"}
	if key.TenantID != "" {
		var tenant models.Tenant
		if err := h.db.Where("id = ? AND status = ?", key.TenantID, "active").First(&tenant).Error; err != nil {
			c.AbortWithStatusJSON(403, gin.H{"error": "tenant is disabled"})
			return true
		}
		nodeAccess = stringListFromJSON(tenant.NodeAccess)
	}
	now := time.Now().UTC()
	_ = h.db.Model(&models.APIKey{}).Where("id = ?", key.ID).Update("last_used_at", &now).Error
	c.Set("actor", key.ID)
	c.Set("authKind", "api")
	c.Set("tenantID", key.TenantID)
	c.Set("role", "api")
	c.Set("scopes", scopes)
	c.Set("nodeAccess", nodeAccess)
	return true
}

func nodeAccessFilter(c *gin.Context) ([]string, bool) {
	raw, exists := c.Get("nodeAccess")
	if !exists {
		return nil, false
	}
	values, ok := raw.([]string)
	if !ok || len(values) == 0 {
		return nil, false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "*" {
			return nil, false
		}
	}
	return values, true
}

func canAccessNode(c *gin.Context, id string) bool {
	allowed, limited := nodeAccessFilter(c)
	if !limited {
		return true
	}
	for _, item := range allowed {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(id)) {
			return true
		}
	}
	return false
}

type quotaPolicy struct {
	MonthlyTrafficQuota uint64
	PerNodeTrafficQuota uint64
	MaxConnections      int
	NodeAccess          []string
}

func (h Handler) quotaPolicyForRequest(c *gin.Context) quotaPolicy {
	role, _ := c.Get("role")
	if strings.EqualFold(strings.TrimSpace(fmt.Sprint(role)), "owner") {
		return quotaPolicy{}
	}
	policy := quotaPolicy{}
	if allowed, limited := nodeAccessFilter(c); limited {
		policy.NodeAccess = allowed
	}
	tenantID := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "tenantID")))
	var tenant models.Tenant
	if tenantID != "" {
		if err := h.db.Where("id = ? AND status = ?", tenantID, "active").First(&tenant).Error; err == nil {
			policy.MonthlyTrafficQuota = tenant.MonthlyTrafficQuota
			policy.PerNodeTrafficQuota = tenant.PerNodeTrafficQuota
			policy.MaxConnections = tenant.MaxConnections
			if len(policy.NodeAccess) == 0 {
				policy.NodeAccess = stringListFromJSON(tenant.NodeAccess)
			}
		}
	}
	authKind := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "authKind")))
	if authKind != "panel" {
		return policy
	}
	actor := strings.TrimSpace(fmt.Sprint(mustContextValue(c, "actor")))
	var user models.PanelUser
	if actor == "" || h.db.Where("id = ? AND status = ?", actor, "active").First(&user).Error != nil {
		return policy
	}
	policy.MonthlyTrafficQuota = minPositiveQuota(policy.MonthlyTrafficQuota, user.MonthlyTrafficQuota)
	policy.PerNodeTrafficQuota = minPositiveQuota(policy.PerNodeTrafficQuota, user.PerNodeTrafficQuota)
	policy.MaxConnections = minPositiveInt(policy.MaxConnections, user.MaxConnections)
	if userAccess := stringListFromJSON(user.NodeAccess); len(userAccess) > 0 {
		policy.NodeAccess = userAccess
	}
	return policy
}

func (h Handler) quotaBlockReason(c *gin.Context) string {
	policy := h.quotaPolicyForRequest(c)
	if policy.MonthlyTrafficQuota == 0 && policy.PerNodeTrafficQuota == 0 && policy.MaxConnections == 0 {
		return ""
	}
	total, connections, perNode := h.quotaUsage(policy.NodeAccess)
	if policy.MonthlyTrafficQuota > 0 && total >= policy.MonthlyTrafficQuota {
		return "monthly traffic quota exceeded"
	}
	if policy.MaxConnections > 0 && connections >= policy.MaxConnections {
		return "connection quota exceeded"
	}
	if policy.PerNodeTrafficQuota > 0 {
		for _, used := range perNode {
			if used >= policy.PerNodeTrafficQuota {
				return "per-node traffic quota exceeded"
			}
		}
	}
	return ""
}

func (h Handler) quotaUsage(nodeAccess []string) (uint64, int, map[string]uint64) {
	var samples []models.NodeTrafficSample
	query := h.db.Order("collected_at desc").Limit(5000)
	nodeAccess = compactStringList(nodeAccess)
	if len(nodeAccess) > 0 && !containsWildcard(nodeAccess) {
		query = query.Where("node_id IN ? OR agent_id IN ?", nodeAccess, nodeAccess)
	}
	if err := query.Find(&samples).Error; err != nil {
		return 0, 0, map[string]uint64{}
	}
	latest := map[string]models.NodeTrafficSample{}
	for _, sample := range samples {
		if _, ok := latest[sample.NodeID]; !ok {
			latest[sample.NodeID] = sample
		}
	}
	perNode := map[string]uint64{}
	var total uint64
	var connections int
	for nodeID, sample := range latest {
		used := sample.RxBytes + sample.TxBytes
		perNode[nodeID] = used
		total += used
		connections += sample.Connections
	}
	return total, connections, perNode
}

func authorizePanelRequest(c *gin.Context) bool {
	if role, _ := c.Get("role"); strings.EqualFold(strings.TrimSpace(fmt.Sprint(role)), "owner") {
		return true
	}
	required := "panel:read"
	switch c.Request.Method {
	case "GET", "HEAD", "OPTIONS":
		required = "panel:read"
	default:
		required = "panel:write"
	}
	raw, exists := c.Get("scopes")
	if !exists {
		return false
	}
	scopes, ok := raw.([]string)
	if !ok {
		return false
	}
	return scopeAllows(scopes, required)
}

func mustContextValue(c *gin.Context, key string) any {
	value, _ := c.Get(key)
	return value
}

func minPositiveQuota(current, next uint64) uint64 {
	if current == 0 {
		return next
	}
	if next == 0 {
		return current
	}
	if next < current {
		return next
	}
	return current
}

func minPositiveInt(current, next int) int {
	if current == 0 {
		return next
	}
	if next == 0 {
		return current
	}
	if next < current {
		return next
	}
	return current
}

func scopeAllows(scopes []string, required string) bool {
	for _, scope := range scopes {
		normalized := strings.ToLower(strings.TrimSpace(scope))
		switch normalized {
		case "*", strings.ToLower(required):
			return true
		case "panel:*":
			if strings.HasPrefix(required, "panel:") {
				return true
			}
		case "panel:write":
			if required == "panel:read" {
				return true
			}
		}
	}
	return false
}

func stringListFromJSON(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		return compactStringList(values)
	}
	var anyValues []any
	if err := json.Unmarshal(raw, &anyValues); err != nil {
		return nil
	}
	out := make([]string, 0, len(anyValues))
	for _, value := range anyValues {
		if text, ok := value.(string); ok {
			out = append(out, text)
		}
	}
	return compactStringList(out)
}

func compactStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
