package all

import (
	"testing"

	"github.com/polagonow/pola/internal/generators"
)

// TestRegisterComplete guards against a generator being dropped from the
// explicit list — a missing entry would otherwise fail silently at runtime.
func TestRegisterComplete(t *testing.T) {
	Register()
	if got, want := len(generators.All()), 16; got != want {
		t.Errorf("registered %d generators, want %d — keep the list in register.go in sync", got, want)
	}
	for _, name := range []string{
		"action", "docs", "dto", "js:bridge", "mailer", "mcp", "migration",
		"model", "page", "repository", "route", "scaffold", "seed", "service",
		"storage", "zod",
	} {
		if _, err := generators.Get(name); err != nil {
			t.Errorf("generator %q not registered: %v", name, err)
		}
	}
}
