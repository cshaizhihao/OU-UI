package hysteria2

import (
	"context"
	"fmt"
	"strings"

	"github.com/cshaizhihao/OU-UI/internal/deploy"
	"github.com/cshaizhihao/OU-UI/internal/provider"
)

type Provider struct{}

var _ provider.Provider = Provider{}

var hysteriaRuntime = deploy.RuntimeManager{
	RuntimeName:   "hysteria2",
	BinaryNames:   []string{"hysteria", "hysteria2"},
	ServicePrefix: "ou-ui-hysteria2",
	ConfigExt:     "yaml",
	ServiceMode:   deploy.ServiceModeManagedNode,
	CommandArgs:   []string{"server", "-c", "{configPath}"},
}

func NewProvider() Provider {
	return Provider{}
}

func (Provider) Name() provider.Runtime {
	return provider.RuntimeHysteria2
}

func (Provider) Validate(spec provider.NodeSpec) error {
	if spec.Runtime != "" && spec.Runtime != provider.RuntimeHysteria2 {
		return fmt.Errorf("hysteria2 provider cannot handle runtime %q", spec.Runtime)
	}
	protocol := strings.ToLower(strings.TrimSpace(spec.Protocol))
	if protocol != "" && protocol != "hysteria2" && protocol != "hy2" {
		return fmt.Errorf("hysteria2 provider cannot handle protocol %q", spec.Protocol)
	}

	cfg, err := ConfigFromNode(spec.Listen, spec.Port, spec.Settings)
	if err != nil {
		return err
	}

	return ValidateConfig(cfg)
}

func (p Provider) Render(spec provider.NodeSpec) ([]byte, error) {
	if err := p.Validate(spec); err != nil {
		return nil, err
	}

	cfg, err := ConfigFromNode(spec.Listen, spec.Port, spec.Settings)
	if err != nil {
		return nil, err
	}

	return RenderYAML(cfg), nil
}

func (Provider) Apply(ctx context.Context, spec provider.NodeSpec) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cfg, err := ConfigFromNode(spec.Listen, spec.Port, spec.Settings)
	if err != nil {
		return err
	}

	return ValidateConfig(cfg)
}

func (Provider) Remove(ctx context.Context, nodeID string) error {
	return ctx.Err()
}

func (Provider) Install(ctx context.Context, req provider.DeployRequest) (provider.StageResult, error) {
	return hysteriaRuntime.Install(ctx, req)
}

func (Provider) ApplyConfig(ctx context.Context, req provider.DeployRequest) (provider.ApplyResult, error) {
	return hysteriaRuntime.ApplyConfig(ctx, req)
}

func (Provider) Reload(ctx context.Context, req provider.DeployRequest) (provider.StageResult, error) {
	return hysteriaRuntime.Reload(ctx, req)
}

func (Provider) Health(ctx context.Context, req provider.DeployRequest) (provider.HealthResult, error) {
	return hysteriaRuntime.Health(ctx, req)
}

func (Provider) Rollback(ctx context.Context, req provider.RollbackRequest) (provider.StageResult, error) {
	return hysteriaRuntime.Rollback(ctx, req)
}
