package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
		"trojan://secret@sg.example.com:443?security=tls#Singapore%20Trojan\n"

	nodes := parseExternalNodes("sub_test", content)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Name != "Tokyo VMess" || nodes[0].Protocol != "vmess" || nodes[0].Address != "jp.example.com" || nodes[0].Port != 443 {
		t.Fatalf("unexpected vmess node: %+v", nodes[0])
	}
	if nodes[1].Name != "Singapore Trojan" || nodes[1].Protocol != "trojan" || nodes[1].Address != "sg.example.com" || nodes[1].Port != 443 {
		t.Fatalf("unexpected trojan node: %+v", nodes[1])
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
