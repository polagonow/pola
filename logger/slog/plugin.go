package slog

import "github.com/polagonow/pola/core"

// Plugin returns the slog logger plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "slog",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Logger](r, New())
		},
	}
}
