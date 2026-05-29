package provider

import "context"

type Runtime string

const (
	RuntimeXray      Runtime = "xray"
	RuntimeSingBox   Runtime = "sing-box"
	RuntimeHysteria2 Runtime = "hysteria2"
)

type NodeSpec struct {
	Runtime  Runtime        `json:"runtime"`
	Protocol string         `json:"protocol"`
	Listen   string         `json:"listen"`
	Port     int            `json:"port"`
	Settings map[string]any `json:"settings"`
}

type Provider interface {
	Name() Runtime
	Validate(spec NodeSpec) error
	Render(spec NodeSpec) ([]byte, error)
	Apply(ctx context.Context, spec NodeSpec) error
	Remove(ctx context.Context, nodeID string) error
}
