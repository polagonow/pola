package nextjs

import "github.com/polagonow/pola/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		Discoverer:     func() framework.Discoverer { return &Discoverer{} },
		RouteBuilder:   func() framework.RouteBuilder { return &RouteBuilder{} },
		EntryGenerator: func() framework.ServerEntryGenerator { return &ReactRSCEntryGenerator{} },
		ClientEntry:    "@pola/react/Client",
	})
}
