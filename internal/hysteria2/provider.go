package hysteria2

import (
	"context"
	"fmt"

	"github.com/cshaizhihao/OU-UI/internal/provider"
)

type Provider struct{}

var _ provider.Provider = Provider{}

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
