package react

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/shell"
)

func init() {
	core.RegisterRenderer(func() core.Renderer { return New() })
	core.RegisterHTMLShell(func() core.HTMLShell {
		return shell.New(`<div id="root"></div>`)
	})
}
