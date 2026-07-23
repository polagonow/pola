package cli

import (
	"testing"

	generatorsall "github.com/polagonow/pola/internal/generators/all"
)

// TestGeneratorSubcommandsAttached guards the Execute-time wiring: generator
// subcommands must be attached after the registry is populated explicitly —
// attaching them in an init() would run against an empty registry and every
// `pola generate <x>` would silently fall through to the bridge generator.
func TestGeneratorSubcommandsAttached(t *testing.T) {
	generatorsall.Register()
	addGeneratorCommands()
	attached := map[string]bool{}
	for _, c := range generateCmd.Commands() {
		attached[c.Name()] = true
	}
	for _, name := range []string{"scaffold", "model", "migration", "repository", "service"} {
		if !attached[name] {
			t.Errorf("pola generate %s subcommand not attached", name)
		}
	}
}
