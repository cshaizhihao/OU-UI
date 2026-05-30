package xray

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMergeRoutingConfigInjectsRoutingAndPreservesExistingOutbounds(t *testing.T) {
	config := []byte(`{
  "inbounds": [{"tag": "in"}],
  "outbounds": [
    {"protocol": "freedom", "tag": "direct"},
    {"protocol": "blackhole", "tag": "blocked"},
    {"protocol": "vmess", "tag": "OU-Auto", "settings": {"vnext": []}}
  ]
}`)
	routing := []byte(`{
  "domainStrategy": "IPIfNonMatch",
  "rules": [
    {"type": "field", "domain": ["geosite:category-ads-all"], "outboundTag": "blocked"},
    {"type": "field", "domain": ["domain:stream.example"], "outboundTag": "OU-Auto"}
  ]
}`)

	merged, err := MergeRoutingConfig(config, routing)
	if err != nil {
		t.Fatalf("merge routing: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(merged, &doc); err != nil {
		t.Fatalf("merged config is invalid json: %v\n%s", err, merged)
	}
	if _, ok := doc["routing"].(map[string]any); !ok {
		t.Fatalf("expected routing document in merged config: %+v", doc)
	}
	outbounds := doc["outbounds"].([]any)
	if !hasXrayOutboundTag(outbounds, "OU-Auto") {
		t.Fatalf("expected existing OU-Auto outbound to be preserved: %+v", outbounds)
	}
}

func TestMergeRoutingConfigRejectsMissingProxyOutboundTag(t *testing.T) {
	config := []byte(`{"outbounds": [{"protocol": "freedom", "tag": "direct"}]}`)
	routing := []byte(`{"rules": [{"type": "field", "domain": ["domain:stream.example"], "outboundTag": "OU-Auto"}]}`)

	_, err := MergeRoutingConfig(config, routing)
	if err == nil || !strings.Contains(err.Error(), "OU-Auto") {
		t.Fatalf("expected missing outbound tag error, got %v", err)
	}
}

func TestMergeRoutingConfigEnsuresDirectAndBlockedOutbounds(t *testing.T) {
	config := []byte(`{"inbounds": [{"tag": "in"}]}`)
	routing := []byte(`{"rules": [{"type": "field", "ip": ["geoip:private"], "outboundTag": "direct"}, {"type": "field", "protocol": ["bittorrent"], "outboundTag": "blocked"}]}`)

	merged, err := MergeRoutingConfig(config, routing)
	if err != nil {
		t.Fatalf("merge routing: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(merged, &doc); err != nil {
		t.Fatalf("merged config is invalid json: %v", err)
	}
	outbounds := doc["outbounds"].([]any)
	if !hasXrayOutboundTag(outbounds, "direct") || !hasXrayOutboundTag(outbounds, "blocked") {
		t.Fatalf("expected direct and blocked outbounds, got %+v", outbounds)
	}
}

func hasXrayOutboundTag(outbounds []any, tag string) bool {
	for _, item := range outbounds {
		outbound, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if outbound["tag"] == tag {
			return true
		}
	}
	return false
}
