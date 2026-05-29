package provider

import (
	"context"
	"errors"
	"fmt"
)

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

type Registry struct {
	providers map[Runtime]Provider
}

func NewRegistry(providers ...Provider) Registry {
	registry := Registry{providers: map[Runtime]Provider{}}
	for _, p := range providers {
		registry.providers[p.Name()] = p
	}
	return registry
}

func (r Registry) Get(runtime Runtime) (Provider, bool) {
	p, ok := r.providers[runtime]
	return p, ok
}

func (r Registry) Render(spec NodeSpec) ([]byte, error) {
	p, ok := r.Get(spec.Runtime)
	if !ok {
		return nil, fmt.Errorf("unsupported runtime %q", spec.Runtime)
	}
	if err := p.Validate(spec); err != nil {
		return nil, err
	}
	return p.Render(spec)
}

func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func RequireString(settings map[string]any, key string) (string, error) {
	value, ok := settings[key]
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return "", fmt.Errorf("%s must be a non-empty string", key)
	}
	return text, nil
}

func OptionalString(settings map[string]any, key string, fallback string) string {
	value, ok := settings[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return fallback
	}
	return text
}

func OptionalBool(settings map[string]any, key string, fallback bool) bool {
	value, ok := settings[key]
	if !ok {
		return fallback
	}
	boolValue, ok := value.(bool)
	if !ok {
		return fallback
	}
	return boolValue
}

func EnsureSettings(spec NodeSpec) map[string]any {
	if spec.Settings == nil {
		return map[string]any{}
	}
	return spec.Settings
}

var ErrUnsupportedProtocol = errors.New("unsupported protocol")
