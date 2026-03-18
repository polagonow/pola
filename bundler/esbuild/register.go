package esbuild

import "github.com/polagonow/pola/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		Bundler: func() framework.Bundler { return &Bundler{} },
	})
}
