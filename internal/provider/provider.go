package provider

import (
	"context"
	"errors"
	"fmt"
	"time"
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

type DeployStageName string

const (
	DeployStageRender   DeployStageName = "render"
	DeployStageInstall  DeployStageName = "install"
	DeployStageApply    DeployStageName = "apply"
	DeployStageReload   DeployStageName = "reload"
	DeployStageHealth   DeployStageName = "health"
	DeployStageRollback DeployStageName = "rollback"
)

type DeployRequest struct {
	NodeID   string
	Spec     NodeSpec
	Rendered []byte
	DataDir  string
	Revision string
	Runner   CommandRunner
}

type RollbackRequest struct {
	NodeID     string
	Spec       NodeSpec
	DataDir    string
	Revision   string
	ConfigPath string
	BackupPath string
	ConfigDir   string
	UnitPath    string
	ServiceName string
	Runner     CommandRunner
}

type StageResult struct {
	Name       DeployStageName `json:"name"`
	Status     string          `json:"status"`
	Message    string          `json:"message,omitempty"`
	StartedAt  string          `json:"startedAt,omitempty"`
	FinishedAt string          `json:"finishedAt,omitempty"`
	DurationMs int64           `json:"durationMs,omitempty"`
}

type ApplyResult struct {
	StageResult
	ConfigPath        string `json:"configPath,omitempty"`
	ConfigDir         string `json:"configDir,omitempty"`
	BackupPath        string `json:"backupPath,omitempty"`
	UnitPath          string `json:"unitPath,omitempty"`
	ServiceName       string `json:"serviceName,omitempty"`
	ServiceMode       string `json:"serviceMode,omitempty"`
	ManagedByOUUI     bool   `json:"managedByOuui"`
	RollbackAvailable bool   `json:"rollbackAvailable"`
}

type HealthResult struct {
	StageResult
	OK             bool   `json:"ok"`
	ServiceName    string `json:"serviceName,omitempty"`
	ServiceStatus  string `json:"serviceStatus,omitempty"`
	ConfigDir      string `json:"configDir,omitempty"`
	UnitPath       string `json:"unitPath,omitempty"`
	ServiceMode    string `json:"serviceMode,omitempty"`
	ManagedByOUUI  bool   `json:"managedByOuui"`
	Runtime        string `json:"runtime,omitempty"`
	RuntimeVersion string `json:"runtimeVersion,omitempty"`
	Listen         string `json:"listen,omitempty"`
	Port           int    `json:"port,omitempty"`
}

type CommandResult struct {
	Command  []string
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (CommandResult, error)
	LookPath(file string) (string, error)
}

type DeploymentProvider interface {
	Install(ctx context.Context, req DeployRequest) (StageResult, error)
	ApplyConfig(ctx context.Context, req DeployRequest) (ApplyResult, error)
	Reload(ctx context.Context, req DeployRequest) (StageResult, error)
	Health(ctx context.Context, req DeployRequest) (HealthResult, error)
	Rollback(ctx context.Context, req RollbackRequest) (StageResult, error)
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
