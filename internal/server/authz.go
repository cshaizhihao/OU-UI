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
