package v8go

import "github.com/polagonow/pola/core"

// Plugin returns the V8 (v8go) JS engine plugin. It mirrors the goja engine
// plugin: the engine is provided empty here and compiles the server bundle
// later via NewSSRPool (core.SSRPoolFactory), called by the pipeline.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "v8go",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.JSEngine](r, &Engine{})
		},
	}
}
