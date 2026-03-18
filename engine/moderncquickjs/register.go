//go:build moderncquickjs

package moderncquickjs

import "github.com/polagonow/pola/core"

// Registered is true when the moderncquickjs build tag is active.
var Registered = true

func init() {
	core.RegisterEngine(func() core.JSEngine { return NewEngine() })
}
