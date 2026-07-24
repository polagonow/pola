package all

import (
	"testing"

	"github.com/polagonow/pola/internal/autoload"
)

// TestRegisterComplete guards against an autoload stage being dropped from the
// explicit list — low-priority stages would otherwise fail silently at runtime.
func TestRegisterComplete(t *testing.T) {
	Register()
	if got, want := len(autoload.All()), 10; got != want {
		t.Errorf("registered %d autoloads, want %d — keep the list in register.go in sync", got, want)
	}
	for _, name := range []string{
		"actionbridge", "dbembed", "dbseed", "embed", "entclient",
		"mcp", "pluginimports", "repos", "routes", "services",
	} {
		if autoload.Get(name) == nil {
			t.Errorf("autoload %q not registered", name)
		}
	}
}
