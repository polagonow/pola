package nextjs

import "gojsx/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		Discoverer:     func() framework.Discoverer { return &NextJSDiscoverer{} },
		RouteBuilder:   func() framework.RouteBuilder { return &NextJSRouteBuilder{} },
		EntryGenerator: func() framework.ServerEntryGenerator { return &ReactRSCEntryGenerator{} },
		ClientEntry:    "@gojsx/react-renderer/client",
	})
}
