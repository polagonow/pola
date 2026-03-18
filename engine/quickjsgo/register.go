//go:build quickjsgo

package quickjsgo

import "github.com/polagonow/pola/core"

// Registered is true when the quickjsgo build tag is active.
var Registered = true

func init() {
	core.RegisterEngine(func() core.JSEngine { return NewEngine() })
}
