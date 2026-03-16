// Package globals installs process and performance globals for the V8 VM.
package globals

import (
	"fmt"

	v8 "rogchap.com/v8go"
)

const src = `
globalThis.process = { env: { NODE_ENV: "production" } };
globalThis.performance = { now: function() { return 0; } };
`

// Enable installs process and performance as globals into ctx.
func Enable(ctx *v8.Context) error {
	if _, err := ctx.RunScript(src, "globals.js"); err != nil {
		return fmt.Errorf("globals: %w", err)
	}
	return nil
}
