package xray

import (
	"encoding/json"
	"fmt"
	"strings"
)

func MergeRoutingConfig(configContent []byte, routingContent []byte) ([]byte, error) {
	if len(routingContent) == 0 || !json.Valid(routingContent) {
		return nil, fmt.Errorf("xray routing document is required")
	}
	var routing map[string]any
	if err := json.Unmarshal(routingContent, &routing); err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(configContent, &doc); err != nil {
		return nil, err
	}
	outbounds, err := ensureRoutingOutbounds(doc["outbounds"], routing)
	if err != nil {
		return nil, err
	}
	doc["routing"] = routing
	doc["outbounds"] = outbounds
	return json.MarshalIndent(doc, "", "  ")
}

func ensureRoutingOutbounds(raw any, routing map[string]any) ([]map[string]any, error) {
	outbounds := normalizeOutbounds(raw)
	existing := map[string]bool{}
	for _, outbound := range outbounds {
		if tag, ok := outbound["tag"].(string); ok && strings.TrimSpace(tag) != "" {
			existing[strings.TrimSpace(tag)] = true
		}
	}
	for _, tag := range routingOutboundTags(routing) {
		switch tag {
		case "direct":
			if !existing[tag] {
				outbounds = append(outbounds, map[string]any{"protocol": "freedom", "tag": tag})
				existing[tag] = true
			}
		case "blocked":
			if !existing[tag] {
				outbounds = append(outbounds, map[string]any{"protocol": "blackhole", "tag": tag})
				existing[tag] = true
			}
		default:
			if !existing[tag] {
				return nil, fmt.Errorf("xray routing outbound tag %q is not present in config", tag)
			}
		}
	}
	if !existing["direct"] {
		outbounds = append(outbounds, map[string]any{"protocol": "freedom", "tag": "direct"})
	}
	if !existing["blocked"] {
		outbounds = append(outbounds, map[string]any{"protocol": "blackhole", "tag": "blocked"})
	}
	return outbounds, nil
}

func normalizeOutbounds(raw any) []map[string]any {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		outbound, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, outbound)
	}
	return out
}

func routingOutboundTags(routing map[string]any) []string {
	rules, ok := routing["rules"].([]any)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	tags := []string{}
	for _, item := range rules {
		rule, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tag, _ := rule["outboundTag"].(string)
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	return tags
}
