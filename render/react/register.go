package react

import (
	"gojsx/framework"
	"gojsx/framework/contract"
	_ "gojsx/render/react/discovery/nextjs"
	_ "gojsx/render/react/shell"
)

func init() {
	framework.RegisterDefaults(framework.Defaults{
		ManifestEncoder: func() framework.ManifestEncoder { return &WebpackManifestEncoder{} },
		StreamProtocol:  func() framework.StreamProtocol { return &RSCFlightProtocol{} },
		RendererFactory: func(pool framework.VMPool, protocol framework.StreamProtocol, bridge contract.BridgeConfig) framework.Renderer {
			return NewVMRenderer(pool, protocol, bridge)
		},
	})
}
