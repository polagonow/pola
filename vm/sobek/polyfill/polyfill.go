// Package polyfill provides Web API polyfills for the Sobek VM.
package polyfill

import (
	"fmt"

	sobeklib "github.com/grafana/sobek"

	"gojsx/vm/polyfill"
)

type runner struct{ rt *sobeklib.Runtime }

func (r *runner) RunScript(src, name string) error {
	if _, err := r.rt.RunString(src); err != nil {
		return fmt.Errorf("polyfill %s: %w", name, err)
	}
	return nil
}

// Enable installs all polyfills onto rt as globals.
// Must be called before rt.RunProgram(serverBundle).
func Enable(rt *sobeklib.Runtime) error {
	return polyfill.LoadAll(&runner{rt})
}
