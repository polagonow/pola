package esbuild

import "gojsx/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		Bundler: func() framework.Bundler { return &Bundler{} },
	})
}
