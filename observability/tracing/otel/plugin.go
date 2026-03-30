package otel

import "github.com/polagonow/pola/core"

// Plugin returns the OpenTelemetry tracing plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "otel",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Tracer](r, New())
		},
	}
}
