package xray

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cshaizhihao/OU-UI/internal/provider"
)

func TestProviderRenderVLESSReality(t *testing.T) {
	spec := provider.NodeSpec{
		Runtime:  provider.RuntimeXray,
		Protocol: ProtocolVLESS,
		Listen:   "0.0.0.0",
		Port:     443,
		Settings: map[string]any{
			"id":                  "node-443",
			"uuid":                "00000000-0000-0000-0000-000000000000",
			"flow":                "xtls-rprx-vision",
			"reality":             true,
			"reality.dest":        "www.example.com:443",
			"reality.serverNames": []any{"www.example.com"},
			"reality.privateKey":  "REPLACE_WITH_GENERATED_PRIVATE_KEY",
			"reality.shortIds":    "0123456789abcdef",
		},
	}

	rendered, err := (Provider{}).Render(spec)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(rendered, &doc); err != nil {
		t.Fatalf("rendered config is not JSON: %v", err)
	}

	text := string(rendered)
	for _, want := range []string{
		`"protocol": "vless"`,
		`"security": "reality"`,
		`"decryption": "none"`,
		`"privateKey": "REPLACE_WITH_GENERATED_PRIVATE_KEY"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered config missing %s:\n%s", want, text)
		}
	}
}

func TestProviderValidateRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name string
		spec provider.NodeSpec
	}{
		{
			name: "bad port",
			spec: provider.NodeSpec{Runtime: provider.RuntimeXray, Protocol: ProtocolVLESS, Port: 70000, Settings: map[string]any{"uuid": "u"}},
		},
		{
			name: "bad protocol",
			spec: provider.NodeSpec{Runtime: provider.RuntimeXray, Protocol: "socks", Port: 1080},
		},
		{
			name: "missing shadowsocks password",
			spec: provider.NodeSpec{Runtime: provider.RuntimeXray, Protocol: ProtocolShadowsocks, Port: 8388, Settings: map[string]any{"method": "aes-128-gcm"}},
		},
		{
			name: "reality on trojan",
			spec: provider.NodeSpec{Runtime: provider.RuntimeXray, Protocol: ProtocolTrojan, Port: 443, Settings: map[string]any{"password": "example", "reality": true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := (Provider{}).Validate(tt.spec); err == nil {
				t.Fatal("Validate() error = nil, want non-nil")
			}
		})
	}
}
