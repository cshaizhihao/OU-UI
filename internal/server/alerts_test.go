package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/agentruntime"
	"github.com/cshaizhihao/OU-UI/internal/config"
	"github.com/cshaizhihao/OU-UI/internal/models"
	"gorm.io/datatypes"
)

func TestHeartbeatTrafficOverloadDeliversGenericWebhook(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	const agentToken = "agent-alert-token"
	if err := db.Create(&models.Agent{
		ID:            "agt_alert",
		Name:          "Alert Agent",
		Status:        models.AgentStatusOnline,
		AuthStatus:    models.AgentAuthActive,
		LastSeenAt:    &now,
		AgentTokenSHA: hashSecret(agentToken),
	}).Error; err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	received := make(chan map[string]any, 4)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode webhook body: %v", err)
		}
		received <- body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer sink.Close()
	if err := db.Create(&models.WebhookEndpoint{
		ID:         "whk_alert",
		Name:       "Alert sink",
		Kind:       "generic",
		URL:        sink.URL,
		Enabled:    true,
		EventTypes: datatypes.JSON(`["agent.traffic.overload","node.traffic.overload"]`),
	}).Error; err != nil {
		t.Fatalf("seed webhook: %v", err)
	}
	router := NewRouter(config.ServerConfig{SecurePath: "/ou-ui", JWTSecret: "test-secret", AgentOfflineAfterSeconds: 45}, db)
	payload, _ := json.Marshal(heartbeatRequest{
		Status: models.AgentStatusOnline,
		Metrics: agentruntime.RuntimeMetrics{
			CPUPercent:   20,
			NetRxRateBps: 180 << 20,
			NetTxRateBps: 12 << 20,
			NodeTraffic: []agentruntime.NodeTrafficMetric{
				{NodeID: "nod_hot", RxRateBps: 96 << 20, TxRateBps: 8 << 20, Connections: 42},
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/ou-ui/api/v1/agents/agt_alert/heartbeat", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+agentToken)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected heartbeat accepted, got %d: %s", resp.Code, resp.Body.String())
	}
	body := waitWebhookBody(t, received)
	event, ok := body["event"].(map[string]any)
	if !ok {
		t.Fatalf("expected generic webhook event payload, got %+v", body)
	}
	if event["eventType"] != "agent.traffic.overload" {
		t.Fatalf("expected agent traffic overload event, got %+v", event)
	}
	var alerts []models.AlertEvent
	if err := db.Order("created_at asc").Find(&alerts).Error; err != nil {
		t.Fatalf("query alerts: %v", err)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected agent and node traffic alerts, got %d: %+v", len(alerts), alerts)
	}
	if alerts[0].EventType != "agent.traffic.overload" || alerts[1].EventType != "node.traffic.overload" {
		t.Fatalf("unexpected alert sequence: %+v", alerts)
	}
	if !alerts[0].Delivered || alerts[0].DeliveredAt == nil {
		t.Fatalf("expected first alert delivered, got %+v", alerts[0])
	}
}

func TestOfflineAgentSweepDeliversServerChanWebhook(t *testing.T) {
	db := openTestDB(t)
	lastSeen := time.Now().UTC().Add(-5 * time.Minute)
	if err := db.Create(&models.Agent{
		ID:         "agt_offline_alert",
		Name:       "Offline Agent",
		Status:     models.AgentStatusOnline,
		AuthStatus: models.AgentAuthActive,
		LastSeenAt: &lastSeen,
	}).Error; err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	received := make(chan map[string]any, 1)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode serverchan body: %v", err)
		}
		received <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer sink.Close()
	if err := db.Create(&models.WebhookEndpoint{
		ID:         "whk_serverchan",
		Name:       "ServerChan sink",
		Kind:       "serverchan",
		URL:        sink.URL,
		Enabled:    true,
		EventTypes: datatypes.JSON(`["agent.offline"]`),
	}).Error; err != nil {
		t.Fatalf("seed webhook: %v", err)
	}
	h := Handler{db: db, cfg: config.ServerConfig{AgentOfflineAfterSeconds: 45}}
	h.markOfflineAgents()
	body := waitWebhookBody(t, received)
	if body["title"] != "OU-UI Alert" {
		t.Fatalf("expected ServerChan title payload, got %+v", body)
	}
	var agent models.Agent
	if err := db.First(&agent, "id = ?", "agt_offline_alert").Error; err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agent.Status != models.AgentStatusOffline {
		t.Fatalf("expected agent marked offline, got %s", agent.Status)
	}
	var alert models.AlertEvent
	if err := db.First(&alert, "event_type = ?", "agent.offline").Error; err != nil {
		t.Fatalf("query offline alert: %v", err)
	}
	if !alert.Delivered || alert.DeliveredAt == nil {
		t.Fatalf("expected offline alert delivered, got %+v", alert)
	}
}

func TestWebhookValidationAndTelegramTestDelivery(t *testing.T) {
	db := openTestDB(t)
	cfg := config.ServerConfig{
		SecurePath:     "/ou-ui",
		AdminUser:      "admin",
		AdminPassword:  "password",
		JWTSecret:      "test-secret",
		AgentJoinToken: "join",
	}
	rawKey := "ouak_webhook_write"
	if err := db.Create(&models.APIKey{
		ID:      "key_webhook_write",
		Name:    "Webhook write",
		KeyHash: hashSecret(rawKey),
		Scopes:  datatypes.JSON(`["panel:read","panel:write"]`),
		Status:  "active",
	}).Error; err != nil {
		t.Fatalf("seed api key: %v", err)
	}
	router := NewRouter(cfg, db)
	badReq := httptest.NewRequest(http.MethodPost, "/ou-ui/api/v1/webhooks", bytes.NewBufferString(`{"name":"Bad Telegram","kind":"telegram"}`))
	badReq.Header.Set("Authorization", "Bearer "+rawKey)
	badReq.Header.Set("Content-Type", "application/json")
	badResp := httptest.NewRecorder()
	router.ServeHTTP(badResp, badReq)
	if badResp.Code != http.StatusBadRequest {
		t.Fatalf("expected telegram validation failure, got %d: %s", badResp.Code, badResp.Body.String())
	}

	received := make(chan map[string]any, 1)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode telegram body: %v", err)
		}
		received <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer sink.Close()
	createBody, _ := json.Marshal(map[string]any{
		"name":       "Telegram sink",
		"kind":       "telegram",
		"url":        sink.URL,
		"secret":     "telegram-token",
		"chatId":     "10086",
		"enabled":    true,
		"eventTypes": []string{"agent.offline", "agent.offline", "cpu.overload"},
	})
	createReq := httptest.NewRequest(http.MethodPost, "/ou-ui/api/v1/webhooks", bytes.NewReader(createBody))
	createReq.Header.Set("Authorization", "Bearer "+rawKey)
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusOK {
		t.Fatalf("expected telegram webhook created, got %d: %s", createResp.Code, createResp.Body.String())
	}
	var hook models.WebhookEndpoint
	if err := json.Unmarshal(createResp.Body.Bytes(), &hook); err != nil {
		t.Fatalf("decode created hook: %v", err)
	}
	var eventTypes []string
	if err := json.Unmarshal(hook.EventTypes, &eventTypes); err != nil {
		t.Fatalf("decode event types: %v", err)
	}
	if len(eventTypes) != 2 {
		t.Fatalf("expected event types to be compacted, got %v", eventTypes)
	}
	testReq := httptest.NewRequest(http.MethodPost, "/ou-ui/api/v1/webhooks/"+hook.ID+"/test", nil)
	testReq.Header.Set("Authorization", "Bearer "+rawKey)
	testResp := httptest.NewRecorder()
	router.ServeHTTP(testResp, testReq)
	if testResp.Code != http.StatusOK {
		t.Fatalf("expected test delivery to pass, got %d: %s", testResp.Code, testResp.Body.String())
	}
	body := waitWebhookBody(t, received)
	if body["chat_id"] != "10086" || body["text"] == "" {
		t.Fatalf("expected telegram payload, got %+v", body)
	}
}

func waitWebhookBody(t *testing.T, received <-chan map[string]any) map[string]any {
	t.Helper()
	select {
	case body := <-received:
		return body
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook delivery")
		return nil
	}
}
