package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/config"
	"github.com/cshaizhihao/OU-UI/internal/models"
	"github.com/cshaizhihao/OU-UI/internal/store"
	"gopkg.in/yaml.v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestParseExternalNodesShareURIs(t *testing.T) {
	vmessBody, _ := json.Marshal(map[string]any{
		"ps":   "Tokyo VMess",
		"add":  "jp.example.com",
		"port": "443",
		"id":   "00000000-0000-0000-0000-000000000001",
	})
	content := "vmess://" + base64.StdEncoding.EncodeToString(vmessBody) + "\n" +
		"trojan://secret@sg.example.com:443?security=tls#Singapore%20Trojan\n" +
		"ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:ss-pass")) + "@hk.example.com:8388#HK%20SS\n" +
		"vless://00000000-0000-0000-0000-000000000003@us.example.com:443?security=reality&pbk=public-key&sid=01#US%20VLESS\n"

	nodes := parseExternalNodes("sub_test", content)
	if len(nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(nodes))
	}
	if nodes[0].Name != "Tokyo VMess" || nodes[0].Protocol != "vmess" || nodes[0].Address != "jp.example.com" || nodes[0].Port != 443 {
		t.Fatalf("unexpected vmess node: %+v", nodes[0])
	}
	if nodes[1].Name != "Singapore Trojan" || nodes[1].Protocol != "trojan" || nodes[1].Address != "sg.example.com" || nodes[1].Port != 443 {
		t.Fatalf("unexpected trojan node: %+v", nodes[1])
	}
	if nodes[2].Name != "HK SS" || nodes[2].Protocol != "ss" || nodes[2].Address != "hk.example.com" || nodes[2].Port != 8388 {
		t.Fatalf("unexpected shadowsocks node: %+v", nodes[2])
	}
	if nodes[3].Name != "US VLESS" || nodes[3].Protocol != "vless" || nodes[3].Address != "us.example.com" || nodes[3].Port != 443 {
		t.Fatalf("unexpected vless node: %+v", nodes[3])
	}
}

func TestParseExternalNodesClashYAML(t *testing.T) {
	content := `
proxies:
  - name: HK SS
    type: ss
    server: hk.example.com
    port: 8388
    cipher: aes-128-gcm
    password: pass
  - {name: US VLESS, type: vless, server: us.example.com, port: 443, uuid: 00000000-0000-0000-0000-000000000002}
`
	nodes := parseExternalNodes("sub_clash", content)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Name != "HK SS" || nodes[0].Protocol != "ss" || nodes[0].Address != "hk.example.com" || nodes[0].Port != 8388 {
		t.Fatalf("unexpected first clash node: %+v", nodes[0])
	}
	if nodes[1].Name != "US VLESS" || nodes[1].Protocol != "vless" || nodes[1].Address != "us.example.com" || nodes[1].Port != 443 {
		t.Fatalf("unexpected second clash node: %+v", nodes[1])
	}
}

func TestParseExternalNodesSingBoxJSON(t *testing.T) {
	content := `{
  "outbounds": [
    {"type": "selector", "tag": "auto", "outbounds": ["HK SS"]},
    {"type": "shadowsocks", "tag": "HK SS", "server": "hk.example.com", "server_port": 8388, "method": "aes-128-gcm", "password": "pass"},
    {"type": "vless", "tag": "US Reality", "server": "us.example.com", "server_port": 443, "uuid": "00000000-0000-0000-0000-000000000004", "tls": {"enabled": true}}
  ]
}`
	nodes := parseExternalNodes("sub_singbox", content)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 sing-box nodes, got %d", len(nodes))
	}
	if nodes[0].Name != "HK SS" || nodes[0].Protocol != "shadowsocks" || nodes[0].Address != "hk.example.com" || nodes[0].Port != 8388 {
		t.Fatalf("unexpected first sing-box node: %+v", nodes[0])
	}
	if nodes[1].Name != "US Reality" || nodes[1].Protocol != "vless" || nodes[1].Address != "us.example.com" || nodes[1].Port != 443 {
		t.Fatalf("unexpected second sing-box node: %+v", nodes[1])
	}
}

func TestGenerateClashYAMLIsParseable(t *testing.T) {
	db := openTestDB(t)
	h := Handler{db: db}
	err := db.Create(&models.ExternalNode{
		ID:       "ext_test",
		Name:     "HK SS",
		Protocol: "shadowsocks",
		Address:  "hk.example.com",
		Port:     8388,
		Config:   datatypes.JSON(`{"cipher":"aes-128-gcm","password":"pass"}`),
		Enabled:  true,
	}).Error
	if err != nil {
		t.Fatalf("seed external node: %v", err)
	}

	content := h.generateClashYAML(clashProfileRequest{
		Name: "Default",
		RuleProviders: []map[string]any{
			{"name": "private", "type": "http", "behavior": "domain", "url": "https://example.com/private.yaml", "interval": 86400},
		},
		RoutingRules: []string{"DOMAIN-SUFFIX,example.com,DIRECT", "MATCH,OU-Auto"},
	})
	var out map[string]any
	if err := yaml.Unmarshal([]byte(content), &out); err != nil {
		t.Fatalf("generated YAML is invalid: %v\n%s", err, content)
	}
	if out["mode"] != "rule" {
		t.Fatalf("expected rule mode, got %#v", out["mode"])
	}
	if _, ok := out["rule-providers"].(map[string]any)["private"]; !ok {
		t.Fatalf("expected private rule provider in %#v", out["rule-providers"])
	}
}

func TestGenerateClashYAMLNormalizesGroupsProvidersAndSelectedNodes(t *testing.T) {
	db := openTestDB(t)
	h := Handler{db: db}
	nodes := []models.ExternalNode{
		{ID: "ext_keep", Name: "Keep SS", Protocol: "shadowsocks", Address: "keep.example.com", Port: 8388, Config: datatypes.JSON(`{"cipher":"aes-128-gcm","password":"pass"}`), Enabled: true},
		{ID: "ext_skip", Name: "Skip SS", Protocol: "shadowsocks", Address: "skip.example.com", Port: 8388, Config: datatypes.JSON(`{"cipher":"aes-128-gcm","password":"pass"}`), Enabled: true},
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatalf("seed external nodes: %v", err)
	}
	content := h.generateClashYAML(clashProfileRequest{
		Name:          "Selected",
		SelectedNodes: []string{"ext_keep"},
		RuleProviders: []map[string]any{
			{"name": "private", "type": "http", "behavior": "domain", "url": "https://example.com/private.yaml", "interval": 86400},
		},
		ProxyGroups: []map[string]any{
			{"name": "Manual", "type": "select", "proxies": []string{"*"}},
		},
	})
	var out map[string]any
	if err := yaml.Unmarshal([]byte(content), &out); err != nil {
		t.Fatalf("generated YAML is invalid: %v\n%s", err, content)
	}
	proxies, ok := out["proxies"].([]any)
	if !ok || len(proxies) != 1 {
		t.Fatalf("expected one selected proxy, got %#v", out["proxies"])
	}
	if strings.Contains(content, "_nodeId") || strings.Contains(content, "Skip SS") {
		t.Fatalf("internal fields or unselected nodes leaked into yaml:\n%s", content)
	}
	if !strings.Contains(content, "RULE-SET,private,Manual") {
		t.Fatalf("expected provider rule to target custom group:\n%s", content)
	}
	if !strings.Contains(content, "proxy-groups:") || !strings.Contains(content, "Keep SS") {
		t.Fatalf("expected normalized proxy group with selected node:\n%s", content)
	}
}

func TestAggregateSubscriptionEndpointServesMultipleFormats(t *testing.T) {
	db := openTestDB(t)
	cfg := config.ServerConfig{
		SecurePath:     "/ou-ui",
		AdminUser:      "admin",
		AdminPassword:  "password",
		JWTSecret:      "test-secret",
		AgentJoinToken: "join",
	}
	const rawKey = "ouak_subscription_read"
	if err := db.Create(&models.APIKey{
		ID:      "key_subscription_read",
		Name:    "Subscription read",
		KeyHash: hashSecret(rawKey),
		Scopes:  datatypes.JSON(`["panel:read"]`),
		Status:  "active",
	}).Error; err != nil {
		t.Fatalf("seed api key: %v", err)
	}
	if err := db.Create(&models.ExternalNode{
		ID:       "ext_aggregate",
		Name:     "HK SS",
		Protocol: "shadowsocks",
		Address:  "hk.example.com",
		Port:     8388,
		Config:   datatypes.JSON(`{"cipher":"aes-128-gcm","password":"pass"}`),
		Enabled:  true,
	}).Error; err != nil {
		t.Fatalf("seed external node: %v", err)
	}
	router := NewRouter(cfg, db)

	clashReq := httptest.NewRequest(http.MethodGet, "/ou-ui/api/v1/subscriptions/aggregate?format=clash", nil)
	clashReq.Header.Set("Authorization", "Bearer "+rawKey)
	clashResp := httptest.NewRecorder()
	router.ServeHTTP(clashResp, clashReq)
	if clashResp.Code != http.StatusOK || !strings.Contains(clashResp.Body.String(), "proxy-groups:") {
		t.Fatalf("expected clash aggregate yaml, got %d: %s", clashResp.Code, clashResp.Body.String())
	}

	v2rayReq := httptest.NewRequest(http.MethodGet, "/ou-ui/api/v1/subscriptions/aggregate?format=v2ray", nil)
	v2rayReq.Header.Set("Authorization", "Bearer "+rawKey)
	v2rayResp := httptest.NewRecorder()
	router.ServeHTTP(v2rayResp, v2rayReq)
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(v2rayResp.Body.String()))
	if v2rayResp.Code != http.StatusOK || err != nil || !strings.Contains(string(decoded), "ss://") {
		t.Fatalf("expected v2ray aggregate shares, got code=%d err=%v body=%s", v2rayResp.Code, err, v2rayResp.Body.String())
	}

	singBoxReq := httptest.NewRequest(http.MethodGet, "/ou-ui/api/v1/subscriptions/aggregate?format=sing-box", nil)
	singBoxReq.Header.Set("Authorization", "Bearer "+rawKey)
	singBoxResp := httptest.NewRecorder()
	router.ServeHTTP(singBoxResp, singBoxReq)
	var singBox map[string]any
	if singBoxResp.Code != http.StatusOK || json.Unmarshal(singBoxResp.Body.Bytes(), &singBox) != nil || singBox["outbounds"] == nil {
		t.Fatalf("expected sing-box aggregate json, got %d: %s", singBoxResp.Code, singBoxResp.Body.String())
	}
}

func TestXrayRoutingConfigCoversGeoSiteGeoIPAndProtocolRules(t *testing.T) {
	db := openTestDB(t)
	h := Handler{db: db}
	rules := []models.RoutingRule{
		{ID: "rte_ads", Name: "Ads", Enabled: true, Priority: 10, RuleType: "ads", Match: "category-ads-all", Action: "block"},
		{ID: "rte_private", Name: "Private", Enabled: true, Priority: 20, RuleType: "geoip", Match: "private", Action: "direct"},
		{ID: "rte_bt", Name: "BT", Enabled: true, Priority: 30, RuleType: "protocol", Match: "bittorrent", Action: "block"},
		{ID: "rte_disabled", Name: "Disabled", Enabled: false, Priority: 1, RuleType: "domain", Match: "disabled.example", Action: "direct"},
	}
	if err := db.Create(&rules).Error; err != nil {
		t.Fatalf("seed routing rules: %v", err)
	}

	config, err := h.xrayRoutingConfig()
	if err != nil {
		t.Fatalf("build xray routing config: %v", err)
	}
	if config["domainStrategy"] != "IPIfNonMatch" {
		t.Fatalf("unexpected domain strategy: %+v", config)
	}
	xrayRules, ok := config["rules"].([]map[string]any)
	if !ok || len(xrayRules) != 3 {
		t.Fatalf("expected three enabled xray rules, got %#v", config["rules"])
	}
	assertRuleField(t, xrayRules[0], "domain", []string{"geosite:category-ads-all"}, "blocked")
	assertRuleField(t, xrayRules[1], "ip", []string{"geoip:private"}, "direct")
	assertRuleField(t, xrayRules[2], "protocol", []string{"bittorrent"}, "blocked")
}

func TestApplyRoutingQueuesOnlyOnlineCapableAgentsWithXrayPayload(t *testing.T) {
	db := openTestDB(t)
	cfg := config.ServerConfig{
		SecurePath:               "/ou-ui",
		AdminUser:                "admin",
		AdminPassword:            "password",
		JWTSecret:                "test-secret",
		AgentJoinToken:           "join",
		AgentOfflineAfterSeconds: 45,
	}
	rawKey := "ouak_write_scope"
	if err := db.Create(&models.APIKey{
		ID:      "key_write",
		Name:    "Write scope",
		KeyHash: hashSecret(rawKey),
		Scopes:  datatypes.JSON(`["panel:write"]`),
		Status:  "active",
	}).Error; err != nil {
		t.Fatalf("seed api key: %v", err)
	}
	now := time.Now().UTC()
	agents := []models.Agent{
		{ID: "agt_ready", Name: "Ready", Status: models.AgentStatusOnline, AuthStatus: models.AgentAuthActive, LastSeenAt: &now, Capabilities: datatypes.JSON(`["task-polling","routing.apply"]`), AgentTokenSHA: hashSecret("ready")},
		{ID: "agt_missing_cap", Name: "Missing Cap", Status: models.AgentStatusOnline, AuthStatus: models.AgentAuthActive, LastSeenAt: &now, Capabilities: datatypes.JSON(`["task-polling"]`), AgentTokenSHA: hashSecret("missing")},
		{ID: "agt_offline", Name: "Offline", Status: models.AgentStatusOffline, AuthStatus: models.AgentAuthActive, LastSeenAt: &now, Capabilities: datatypes.JSON(`["task-polling","routing.apply"]`), AgentTokenSHA: hashSecret("offline")},
	}
	if err := db.Create(&agents).Error; err != nil {
		t.Fatalf("seed agents: %v", err)
	}
	if err := db.Create(&models.RoutingRule{
		ID:       "rte_ads_apply",
		Name:     "Ads",
		Enabled:  true,
		Priority: 10,
		RuleType: "ads",
		Match:    "category-ads-all",
		Action:   "block",
	}).Error; err != nil {
		t.Fatalf("seed routing rule: %v", err)
	}
	router := NewRouter(cfg, db)

	req := httptest.NewRequest(http.MethodPost, "/ou-ui/api/v1/routing/apply", bytes.NewBufferString(`{"agentIds":["agt_ready","agt_missing_cap","agt_offline"]}`))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected apply routing to pass, got %d: %s", resp.Code, resp.Body.String())
	}
	var body struct {
		Count int `json:"count"`
		Tasks []struct {
			AgentID string          `json:"agentId"`
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode apply response: %v", err)
	}
	if body.Count != 1 || len(body.Tasks) != 1 || body.Tasks[0].AgentID != "agt_ready" {
		t.Fatalf("expected one ready agent task, got %+v", body)
	}
	var payload struct {
		Runtime string `json:"runtime"`
		Routing struct {
			DomainStrategy string           `json:"domainStrategy"`
			Rules          []map[string]any `json:"rules"`
		} `json:"routing"`
	}
	if err := json.Unmarshal(body.Tasks[0].Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.Runtime != "xray" || payload.Routing.DomainStrategy != "IPIfNonMatch" || len(payload.Routing.Rules) != 1 {
		t.Fatalf("unexpected routing payload: %+v", payload)
	}
	if payload.Routing.Rules[0]["outboundTag"] != "blocked" {
		t.Fatalf("expected ads rule to block, got %+v", payload.Routing.Rules[0])
	}
}

func TestLoadBalancerEntrySelectsBestHealthyWeightedMember(t *testing.T) {
	db := openTestDB(t)
	cfg := config.ServerConfig{
		SecurePath:               "/ou-ui",
		AdminUser:                "admin",
		AdminPassword:            "password",
		JWTSecret:                "test-secret",
		AgentJoinToken:           "join",
		AgentOfflineAfterSeconds: 45,
	}
	rawKey := "ouak_ha_read"
	if err := db.Create(&models.APIKey{
		ID:      "key_ha_read",
		Name:    "HA read",
		KeyHash: hashSecret(rawKey),
		Scopes:  datatypes.JSON(`["panel:read"]`),
		Status:  "active",
	}).Error; err != nil {
		t.Fatalf("seed api key: %v", err)
	}
	members := []map[string]any{
		{"id": "agt_down", "name": "Down", "address": "down.example.com", "port": 443, "latencyMs": 1, "lossPercent": 0, "weight": 10, "status": "down"},
		{"id": "agt_lossy", "name": "Lossy", "address": "lossy.example.com", "port": 443, "latencyMs": 25, "lossPercent": 7.5, "weight": 1, "status": "healthy"},
		{"id": "agt_best", "name": "Best", "address": "best.example.com", "port": 443, "latencyMs": 55, "lossPercent": 0.1, "weight": 3, "status": "healthy"},
	}
	if err := db.Create(&models.LoadBalancerGroup{
		ID:                  "lbg_edge",
		Name:                "Edge HA",
		EntryTag:            "ou-ha-edge",
		Strategy:            "latency-loss",
		Members:             mustJSON(members),
		Status:              "ready",
		LastDecision:        mustJSON(map[string]any{}),
		HealthCheckInterval: 30,
	}).Error; err != nil {
		t.Fatalf("seed load balancer: %v", err)
	}
	router := NewRouter(cfg, db)

	req := httptest.NewRequest(http.MethodGet, "/ou-ui/api/v1/load-balancers/lbg_edge/entry", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected HA entry response, got %d: %s", resp.Code, resp.Body.String())
	}
	var body struct {
		GroupID   string         `json:"groupId"`
		EntryTag  string         `json:"entryTag"`
		Status    string         `json:"status"`
		Selected  string         `json:"selected"`
		Member    map[string]any `json:"member"`
		Decision  map[string]any `json:"decision"`
		EntryPath string         `json:"entryPath"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode HA entry response: %v", err)
	}
	if body.GroupID != "lbg_edge" || body.EntryTag != "ou-ha-edge" || body.EntryPath != "/ou-ui/api/v1/load-balancers/lbg_edge/entry" {
		t.Fatalf("unexpected HA entry identity: %+v", body)
	}
	if body.Status != "ready" || body.Selected != "agt_best" {
		t.Fatalf("expected agt_best to be selected, got %+v", body)
	}
	if body.Member["address"] != "best.example.com" {
		t.Fatalf("expected selected member details, got %+v", body.Member)
	}
	if body.Decision["selected"] != "agt_best" {
		t.Fatalf("expected decision to match selected member, got %+v", body.Decision)
	}
	var group models.LoadBalancerGroup
	if err := db.First(&group, "id = ?", "lbg_edge").Error; err != nil {
		t.Fatalf("reload load balancer: %v", err)
	}
	if group.Status != "ready" || !bytes.Contains(group.LastDecision, []byte("agt_best")) {
		t.Fatalf("expected persisted decision to be refreshed, got status=%s decision=%s", group.Status, group.LastDecision)
	}
}

func TestLoadBalancerHealthUpdateSwitchesSelectedMember(t *testing.T) {
	db := openTestDB(t)
	cfg := config.ServerConfig{
		SecurePath:               "/ou-ui",
		AdminUser:                "admin",
		AdminPassword:            "password",
		JWTSecret:                "test-secret",
		AgentJoinToken:           "join",
		AgentOfflineAfterSeconds: 45,
	}
	rawKey := "ouak_ha_write"
	if err := db.Create(&models.APIKey{
		ID:      "key_ha_write",
		Name:    "HA write",
		KeyHash: hashSecret(rawKey),
		Scopes:  datatypes.JSON(`["panel:read","panel:write"]`),
		Status:  "active",
	}).Error; err != nil {
		t.Fatalf("seed api key: %v", err)
	}
	members := []map[string]any{
		{"id": "agt_primary", "name": "Primary", "address": "primary.example.com", "port": 443, "latencyMs": 10, "lossPercent": 0, "weight": 1, "status": "healthy"},
		{"id": "agt_backup", "name": "Backup", "address": "backup.example.com", "port": 443, "latencyMs": 80, "lossPercent": 0, "weight": 1, "status": "healthy"},
	}
	if err := db.Create(&models.LoadBalancerGroup{
		ID:                  "lbg_switch",
		Name:                "Switch HA",
		EntryTag:            "ou-ha-switch",
		Strategy:            "latency-loss",
		Members:             mustJSON(members),
		Status:              "ready",
		LastDecision:        mustJSON(map[string]any{}),
		HealthCheckInterval: 30,
	}).Error; err != nil {
		t.Fatalf("seed load balancer: %v", err)
	}
	router := NewRouter(cfg, db)
	payload, _ := json.Marshal(map[string]any{
		"members": []map[string]any{
			{"id": "agt_primary", "status": "down", "latencyMs": 5000, "lossPercent": 100, "lastError": "probe timeout"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/ou-ui/api/v1/load-balancers/lbg_switch/health", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected HA health update, got %d: %s", resp.Code, resp.Body.String())
	}
	var body struct {
		Decision map[string]any `json:"decision"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Decision["selected"] != "agt_backup" {
		t.Fatalf("expected backup to be selected after primary down, got %+v", body.Decision)
	}
	var group models.LoadBalancerGroup
	if err := db.First(&group, "id = ?", "lbg_switch").Error; err != nil {
		t.Fatalf("reload load balancer: %v", err)
	}
	if group.Status != "ready" || !bytes.Contains(group.LastDecision, []byte("agt_backup")) || !bytes.Contains(group.Members, []byte("probe timeout")) {
		t.Fatalf("expected persisted health switch, status=%s members=%s decision=%s", group.Status, group.Members, group.LastDecision)
	}
}

func TestLoadBalancerDecisionSkipsOfflineAgentRecord(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&[]models.Agent{
		{ID: "agt_fast_offline", Name: "Fast Offline", Status: models.AgentStatusOffline, AuthStatus: models.AgentAuthActive, LastSeenAt: &now},
		{ID: "agt_slow_online", Name: "Slow Online", Status: models.AgentStatusOnline, AuthStatus: models.AgentAuthActive, LastSeenAt: &now},
	}).Error; err != nil {
		t.Fatalf("seed agents: %v", err)
	}
	h := Handler{db: db, cfg: config.ServerConfig{AgentOfflineAfterSeconds: 45}}
	decision := h.loadBalancerDecision([]map[string]any{
		{"id": "agt_fast_offline", "latencyMs": 1, "lossPercent": 0, "weight": 10, "status": "healthy"},
		{"id": "agt_slow_online", "latencyMs": 100, "lossPercent": 0, "weight": 1, "status": "healthy"},
	})
	body := mapFromJSON(decision)
	if body["selected"] != "agt_slow_online" {
		t.Fatalf("expected offline agent to be skipped, got %s", decision)
	}
}

func assertRuleField(t *testing.T, rule map[string]any, field string, want []string, outbound string) {
	t.Helper()
	if rule["outboundTag"] != outbound {
		t.Fatalf("expected outbound %q, got %+v", outbound, rule)
	}
	got, ok := rule[field].([]string)
	if !ok {
		t.Fatalf("expected %s string slice, got %+v", field, rule[field])
	}
	if len(got) != len(want) {
		t.Fatalf("expected %s=%v, got %v", field, want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %s=%v, got %v", field, want, got)
		}
	}
}

func TestAPIKeyReadScopeCannotMutate(t *testing.T) {
	db := openTestDB(t)
	cfg := config.ServerConfig{
		SecurePath:     "/ou-ui",
		AdminUser:      "admin",
		AdminPassword:  "password",
		JWTSecret:      "test-secret",
		AgentJoinToken: "join",
	}
	now := time.Now().UTC()
	if err := db.Create(&models.Agent{
		ID:            "agt_scope",
		Name:          "Scope Agent",
		Status:        models.AgentStatusOnline,
		AuthStatus:    models.AgentAuthActive,
		LastSeenAt:    &now,
		Capabilities:  datatypes.JSON(`["task-polling","routing.apply"]`),
		AgentTokenSHA: hashSecret("agent-token"),
	}).Error; err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	const rawKey = "ouak_read_scope"
	if err := db.Create(&models.APIKey{
		ID:      "key_scope",
		Name:    "Read scope",
		KeyHash: hashSecret(rawKey),
		Scopes:  datatypes.JSON(`["panel:read"]`),
		Status:  "active",
	}).Error; err != nil {
		t.Fatalf("seed api key: %v", err)
	}
	router := NewRouter(cfg, db)

	getReq := httptest.NewRequest(http.MethodGet, "/ou-ui/api/v1/agents", nil)
	getReq.Header.Set("Authorization", "Bearer "+rawKey)
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected read to pass, got %d: %s", getResp.Code, getResp.Body.String())
	}

	body := bytes.NewBufferString(`{"name":"Block Ads","ruleType":"ads","match":"category-ads-all","action":"block"}`)
	postReq := httptest.NewRequest(http.MethodPost, "/ou-ui/api/v1/routing/rules", body)
	postReq.Header.Set("Authorization", "Bearer "+rawKey)
	postReq.Header.Set("Content-Type", "application/json")
	postResp := httptest.NewRecorder()
	router.ServeHTTP(postResp, postReq)
	if postResp.Code != http.StatusForbidden {
		t.Fatalf("expected write to be forbidden, got %d: %s", postResp.Code, postResp.Body.String())
	}
}

func TestPanelUserRBACFiltersOverviewAndEnforcesNodeQuota(t *testing.T) {
	db := openTestDB(t)
	cfg := config.ServerConfig{
		SecurePath:               "/ou-ui",
		AdminUser:                "admin",
		AdminPassword:            "password",
		JWTSecret:                "test-secret",
		AgentJoinToken:           "join",
		AgentOfflineAfterSeconds: 45,
	}
	now := time.Now().UTC()
	agents := []models.Agent{
		{ID: "agt_allowed", Name: "Allowed", Status: models.AgentStatusOnline, AuthStatus: models.AgentAuthActive, LastSeenAt: &now, Capabilities: datatypes.JSON(`["task-polling","noop"]`), AgentTokenSHA: hashSecret("allowed")},
		{ID: "agt_denied", Name: "Denied", Status: models.AgentStatusOnline, AuthStatus: models.AgentAuthActive, LastSeenAt: &now, Capabilities: datatypes.JSON(`["task-polling","noop"]`), AgentTokenSHA: hashSecret("denied")},
	}
	if err := db.Create(&agents).Error; err != nil {
		t.Fatalf("seed agents: %v", err)
	}
	if err := db.Create(&models.Node{ID: "nod_allowed", AgentID: "agt_allowed", Name: "Allowed node", Runtime: "xray", Protocol: "vless", Status: "ready"}).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}
	if err := db.Create(&models.NodeTrafficSample{NodeID: "nod_allowed", AgentID: "agt_allowed", RxBytes: 100, TxBytes: 0, Connections: 1, CollectedAt: now}).Error; err != nil {
		t.Fatalf("seed traffic: %v", err)
	}
	if err := db.Create(&models.Tenant{
		ID:                  "ten_rbac",
		Name:                "RBAC",
		Status:              "active",
		Role:                "operator",
		NodeAccess:          datatypes.JSON(`["agt_allowed"]`),
		PerNodeTrafficQuota: 100,
	}).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	if err := db.Create(&models.PanelUser{
		ID:          "usr_rbac",
		TenantID:    "ten_rbac",
		Username:    "subuser",
		PasswordSHA: hashSecret("secret"),
		Role:        "operator",
		Status:      "active",
	}).Error; err != nil {
		t.Fatalf("seed panel user: %v", err)
	}
	router := NewRouter(cfg, db)

	loginReq := httptest.NewRequest(http.MethodPost, "/ou-ui/api/v1/auth/login", bytes.NewBufferString(`{"username":"subuser","password":"secret"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected subuser login, got %d: %s", loginResp.Code, loginResp.Body.String())
	}
	var loginBody struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginBody); err != nil || loginBody.Token == "" {
		t.Fatalf("decode login token: %v %q", err, loginResp.Body.String())
	}

	overviewReq := httptest.NewRequest(http.MethodGet, "/ou-ui/api/v1/overview", nil)
	overviewReq.Header.Set("Authorization", "Bearer "+loginBody.Token)
	overviewResp := httptest.NewRecorder()
	router.ServeHTTP(overviewResp, overviewReq)
	if overviewResp.Code != http.StatusOK {
		t.Fatalf("expected filtered overview, got %d: %s", overviewResp.Code, overviewResp.Body.String())
	}
	var overview map[string]any
	_ = json.Unmarshal(overviewResp.Body.Bytes(), &overview)
	if int(overview["agentsTotal"].(float64)) != 1 || int(overview["nodesTotal"].(float64)) != 1 {
		t.Fatalf("expected overview to be tenant-filtered, got %+v", overview)
	}

	taskReq := httptest.NewRequest(http.MethodPost, "/ou-ui/api/v1/tasks", bytes.NewBufferString(`{"agentId":"agt_allowed","type":"noop"}`))
	taskReq.Header.Set("Authorization", "Bearer "+loginBody.Token)
	taskReq.Header.Set("Content-Type", "application/json")
	taskResp := httptest.NewRecorder()
	router.ServeHTTP(taskResp, taskReq)
	if taskResp.Code != http.StatusForbidden || !strings.Contains(taskResp.Body.String(), "per-node traffic quota exceeded") {
		t.Fatalf("expected per-node quota block, got %d: %s", taskResp.Code, taskResp.Body.String())
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "ou-ui-test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}
