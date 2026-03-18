//go:build goja

package goja

import "github.com/polagonow/pola/core"

// Registered is set to true when the goja build tag is active.
// The pipeline uses this to auto-select Goja when no engine is explicitly configured.
var Registered = true

func init() {
	core.RegisterEngine(func() core.JSEngine { return &Engine{} })
}
