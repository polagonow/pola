package cron

import "github.com/polagonow/pola/core"

func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "task-cron",
		Fn: func(r *core.Registry) {
			// Task scheduler is started by the app lifecycle
		},
	}
}
