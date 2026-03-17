package react

import (
	"gojsx/framework"
	"gojsx/framework/contract"
)

func init() {
	framework.RegisterDefaults(framework.Defaults{
		StreamProtocol: func() framework.StreamProtocol { return &RSCFlightProtocol{} },
		RendererFactory: func(
			pool framework.VMPool, protocol framework.StreamProtocol, bridge contract.BridgeConfig,
		) framework.Renderer {
			return NewVMRenderer(pool, protocol, bridge)
		},
	})
}
