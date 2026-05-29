package providers

import (
	"github.com/cshaizhihao/OU-UI/internal/hysteria2"
	"github.com/cshaizhihao/OU-UI/internal/provider"
	"github.com/cshaizhihao/OU-UI/internal/xray"
)

func DefaultRegistry() provider.Registry {
	return provider.NewRegistry(
		xray.Provider{},
		hysteria2.Provider{},
	)
}
