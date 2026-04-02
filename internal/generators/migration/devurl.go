package migration

import (
	"fmt"

	"github.com/polagonow/pola/polafile"
)

// devURLForAdapter returns an Atlas dev-database URL based on the adapter.
// SQLite uses in-memory. Postgres/MySQL use Atlas Docker integration for zero-config.
func devURLForAdapter(adapter string) (string, error) {
	switch adapter {
	case "sqlite":
		return "sqlite://file?mode=memory&_fk=1", nil
	case "postgresql", "postgres":
		return "docker://postgres/15/dev?search_path=public", nil
	case "mysql":
		return "docker://mysql/8/dev", nil
	default:
		return "", fmt.Errorf("unsupported adapter %q; use --dev-url to specify manually", adapter)
	}
}

// resolveDevURL picks the dev URL with priority: CLI flag > Polafile dev_url > auto-detect from adapter.
func resolveDevURL(flagValue string, pf *polafile.Polafile, env, adapter string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if pf != nil {
		if u := pf.DatabaseDevURL(env); u != "" {
			return u, nil
		}
	}
	return devURLForAdapter(adapter)
}
