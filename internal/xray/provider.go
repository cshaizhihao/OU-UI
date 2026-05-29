package xray

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/cshaizhihao/OU-UI/internal/provider"
)

var supportedProtocols = map[string]struct{}{
	ProtocolVLESS:       {},
	ProtocolVMess:       {},
	ProtocolTrojan:      {},
	ProtocolShadowsocks: {},
}

func (p Provider) Validate(spec provider.NodeSpec) error {
	if spec.Runtime != "" && spec.Runtime != provider.RuntimeXray {
		return fmt.Errorf("xray provider cannot handle runtime %q", spec.Runtime)
	}
	cfg, err := ConfigFromNodeSpec(spec)
	if err != nil {
		return err
	}
	return ValidateConfig(cfg)
}

func (p Provider) Render(spec provider.NodeSpec) ([]byte, error) {
	if err := p.Validate(spec); err != nil {
		return nil, err
	}
	cfg, err := ConfigFromNodeSpec(spec)
	if err != nil {
		return nil, err
	}
	return RenderConfig(cfg)
}

func (Provider) Apply(ctx context.Context, spec provider.NodeSpec) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return (Provider{}).Validate(spec)
}

func (Provider) Remove(ctx context.Context, nodeID string) error {
	return ctx.Err()
}

func ConfigFromNodeSpec(spec provider.NodeSpec) (Config, error) {
	cfg := Config{
		ID:       stringValue(spec.Settings, "id"),
		Protocol: strings.ToLower(strings.TrimSpace(spec.Protocol)),
		Listen:   strings.TrimSpace(spec.Listen),
		Port:     spec.Port,
		Remark:   stringValue(spec.Settings, "remark"),
	}
	if cfg.Listen == "" {
		cfg.Listen = "0.0.0.0"
	}

	switch cfg.Protocol {
	case ProtocolVLESS:
		cfg.VLESS = &VLESSConfig{
			UUID:       firstString(spec.Settings, "uuid", "id"),
			Flow:       stringValue(spec.Settings, "flow"),
			Encryption: defaultString(stringValue(spec.Settings, "encryption"), "none"),
		}
	case ProtocolVMess:
		cfg.VMess = &VMessConfig{
			UUID:     firstString(spec.Settings, "uuid", "id"),
			AlterID:  intValue(spec.Settings, "alterId"),
			Security: defaultString(stringValue(spec.Settings, "security"), "auto"),
		}
	case ProtocolTrojan:
		cfg.Trojan = &TrojanConfig{
			Password: firstString(spec.Settings, "password", "uuid", "id"),
		}
	case ProtocolShadowsocks:
		cfg.SS = &SSConfig{
			Method:   defaultString(stringValue(spec.Settings, "method"), "aes-128-gcm"),
			Password: stringValue(spec.Settings, "password"),
			Network:  defaultString(stringValue(spec.Settings, "network"), "tcp,udp"),
		}
	}

	if boolValue(spec.Settings, "reality") || boolValue(spec.Settings, "reality.enabled") || boolValue(spec.Settings, "realityEnabled") || stringValue(spec.Settings, "reality.dest") != "" {
		cfg.Reality = &Reality{
			Enabled:     true,
			Dest:        firstString(spec.Settings, "reality.dest", "dest"),
			ServerNames: stringSliceValue(spec.Settings, "reality.serverNames", "reality.serverName", "serverNames", "serverName"),
			PrivateKey:  firstString(spec.Settings, "reality.privateKey", "privateKey"),
			PublicKey:   firstString(spec.Settings, "reality.publicKey", "publicKey"),
			ShortIDs:    stringSliceValue(spec.Settings, "reality.shortIds", "reality.shortId", "shortIds", "shortId"),
			SpiderX:     defaultString(firstString(spec.Settings, "reality.spiderX", "spiderX"), "/"),
		}
	}

	return cfg, nil
}

func ValidateConfig(cfg Config) error {
	protocol := strings.ToLower(strings.TrimSpace(cfg.Protocol))
	if _, ok := supportedProtocols[protocol]; !ok {
		return fmt.Errorf("unsupported xray protocol %q", cfg.Protocol)
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if cfg.Listen != "" && net.ParseIP(cfg.Listen) == nil {
		return fmt.Errorf("listen must be a valid IP address")
	}

	switch protocol {
	case ProtocolVLESS:
		if cfg.VLESS == nil || cfg.VLESS.UUID == "" {
			return errors.New("vless requires uuid")
		}
	case ProtocolVMess:
		if cfg.VMess == nil || cfg.VMess.UUID == "" {
			return errors.New("vmess requires uuid")
		}
	case ProtocolTrojan:
		if cfg.Trojan == nil || cfg.Trojan.Password == "" {
			return errors.New("trojan requires password")
		}
	case ProtocolShadowsocks:
		if cfg.SS == nil || cfg.SS.Method == "" || cfg.SS.Password == "" {
			return errors.New("shadowsocks requires method and password")
		}
	}

	if cfg.Reality != nil && cfg.Reality.Enabled {
		if protocol != ProtocolVLESS {
			return errors.New("reality is only supported for vless")
		}
		if cfg.Reality.Dest == "" || cfg.Reality.PrivateKey == "" {
			return errors.New("reality requires dest and privateKey")
		}
		if len(cfg.Reality.ServerNames) == 0 {
			return errors.New("reality requires at least one serverName")
		}
	}
	return nil
}

func RenderConfig(cfg Config) ([]byte, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	protocol := strings.ToLower(strings.TrimSpace(cfg.Protocol))
	inbound := xrayInbound{
		Tag:      defaultString(cfg.ID, "ou-ui-"+protocol+"-"+strconv.Itoa(cfg.Port)),
		Listen:   defaultString(cfg.Listen, "0.0.0.0"),
		Port:     cfg.Port,
		Protocol: protocol,
		Sniffing: &xraySniffing{
			Enabled:      true,
			DestOverride: []string{"http", "tls", "quic"},
		},
	}

	switch protocol {
	case ProtocolVLESS:
		client := map[string]any{
			"id": cfg.VLESS.UUID,
		}
		if cfg.VLESS.Flow != "" {
			client["flow"] = cfg.VLESS.Flow
		}
		inbound.Settings = map[string]any{
			"clients":    []map[string]any{client},
			"decryption": defaultString(cfg.VLESS.Encryption, "none"),
		}
	case ProtocolVMess:
		inbound.Settings = map[string]any{
			"clients": []map[string]any{{
				"id":       cfg.VMess.UUID,
				"alterId":  cfg.VMess.AlterID,
				"security": defaultString(cfg.VMess.Security, "auto"),
			}},
		}
	case ProtocolTrojan:
		inbound.Settings = map[string]any{
			"clients": []map[string]any{{
				"password": cfg.Trojan.Password,
			}},
		}
	case ProtocolShadowsocks:
		inbound.Settings = map[string]any{
			"method":   cfg.SS.Method,
			"password": cfg.SS.Password,
			"network":  defaultString(cfg.SS.Network, "tcp,udp"),
		}
	}

	if cfg.Reality != nil && cfg.Reality.Enabled {
		inbound.StreamSettings = map[string]any{
			"network":  "tcp",
			"security": "reality",
			"realitySettings": map[string]any{
				"show":        false,
				"dest":        cfg.Reality.Dest,
				"xver":        0,
				"serverNames": cfg.Reality.ServerNames,
				"privateKey":  cfg.Reality.PrivateKey,
				"shortIds":    cfg.Reality.ShortIDs,
				"spiderX":     defaultString(cfg.Reality.SpiderX, "/"),
			},
		}
	}

	doc := xrayDocument{
		Log: map[string]string{"loglevel": "warning"},
		Inbounds: []xrayInbound{
			inbound,
		},
		Outbounds: []map[string]any{
			{"protocol": "freedom", "tag": "direct"},
			{"protocol": "blackhole", "tag": "blocked"},
		},
	}
	return json.MarshalIndent(doc, "", "  ")
}

type xrayDocument struct {
	Log       map[string]string `json:"log"`
	Inbounds  []xrayInbound     `json:"inbounds"`
	Outbounds []map[string]any `json:"outbounds"`
}

type xrayInbound struct {
	Tag            string         `json:"tag,omitempty"`
	Listen         string         `json:"listen"`
	Port           int            `json:"port"`
	Protocol       string         `json:"protocol"`
	Settings       map[string]any `json:"settings"`
	StreamSettings map[string]any `json:"streamSettings,omitempty"`
	Sniffing        *xraySniffing  `json:"sniffing,omitempty"`
}

type xraySniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(values, key); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := lookupValue(values, key)
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func intValue(values map[string]any, key string) int {
	if values == nil {
		return 0
	}
	raw, ok := lookupValue(values, key)
	if !ok {
		return 0
	}
	switch typed := raw.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		value, _ := typed.Int64()
		return int(value)
	case string:
		value, _ := strconv.Atoi(strings.TrimSpace(typed))
		return value
	default:
		return 0
	}
}

func boolValue(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	raw, ok := lookupValue(values, key)
	if !ok {
		return false
	}
	switch typed := raw.(type) {
	case bool:
		return typed
	case string:
		value, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return value
	default:
		return false
	}
}

func stringSliceValue(values map[string]any, keys ...string) []string {
	for _, key := range keys {
		raw, ok := lookupValue(values, key)
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case []string:
			return compactStrings(typed)
		case []any:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				out = append(out, fmt.Sprint(item))
			}
			return compactStrings(out)
		case string:
			return compactStrings(strings.Split(typed, ","))
		}
	}
	return nil
}

func lookupValue(values map[string]any, key string) (any, bool) {
	if value, ok := values[key]; ok {
		return value, true
	}
	if !strings.Contains(key, ".") {
		return nil, false
	}
	var current any = values
	for _, part := range strings.Split(key, ".") {
		mapping, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok := mapping[part]
		if !ok {
			return nil, false
		}
		current = value
	}
	return current, true
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
