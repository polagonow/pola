package nativersc

import (
	"fmt"

	"github.com/polagonow/pola/core"
	reactrenderer "github.com/polagonow/pola/renderer/react"
	"github.com/polagonow/pola/shell"
)

// Plugin returns the nativersc renderer plugin. It registers the renderer and an
// HTML shell whose mount point matches the React client (@pola/react/client), so
// the unchanged browser client hydrates the Go-generated Flight payload.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "nativersc",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Renderer](r, New())
			core.ProvideValue[core.HTMLShell](r, shell.New(fmt.Sprintf(`<div id="%s"></div>`, reactrenderer.RootElementID)))
		},
	}
}
