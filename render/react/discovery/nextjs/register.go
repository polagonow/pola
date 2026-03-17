package nextjs

import "gojsx/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		Discoverer:     func() framework.Discoverer { return &Discoverer{} },
		RouteBuilder:   func() framework.RouteBuilder { return &RouteBuilder{} },
		EntryGenerator: func() framework.ServerEntryGenerator { return &ReactRSCEntryGenerator{} },
		ClientEntry:    "@gojsx/react/Client",
	})
}
