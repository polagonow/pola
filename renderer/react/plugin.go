package react

import (
	"fmt"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
	"github.com/polagonow/pola/shell"
)

// Plugin returns the React renderer plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "react",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Renderer](r, New())
			core.ProvideValue[core.HTMLShell](r, shell.New(fmt.Sprintf(`<div id="%s"></div>`, globals.RootElementID)))
		},
	}
}
