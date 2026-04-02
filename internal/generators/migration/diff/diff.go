// Package diff defines the DiffGenerator interface and registry for
// ORM-specific migration diff plugins. Each ORM (ent, gorm) implements
// DiffGenerator to produce versioned SQL migration files by diffing
// model schemas against the migration directory.
package diff

import (
	"context"
	"fmt"
)

// Config holds all configuration needed to generate a migration diff.
type Config struct {
	Name          string // migration name (e.g. "CreateUsers")
	ProjectDir    string // absolute path to project root
	ModulePath    string // Go module path from go.mod
	GoVersion     string // Go version from go.mod (e.g. "1.25.0")
	ModelsDir     string // models directory (relative to project root)
	MigrationsDir string // migrations directory (absolute path)
	DevURL        string // dev database URL for Atlas
	Adapter       string // database adapter (sqlite, postgresql, mysql)
}

// DiffGenerator is implemented by ORM plugins to produce versioned migration
// files by diffing model schemas against the migration directory.
type DiffGenerator interface {
	// Name returns the generator name (e.g. "ent", "gorm").
	Name() string
	// Diff generates a migration by diffing the ORM schema against the
	// migration directory and writes the resulting SQL file.
	Diff(ctx context.Context, cfg Config) error
}

var generators = map[string]DiffGenerator{}

// Register adds a DiffGenerator to the registry.
func Register(g DiffGenerator) {
	generators[g.Name()] = g
}

// Get returns the DiffGenerator for the given ORM name.
func Get(name string) (DiffGenerator, error) {
	g, ok := generators[name]
	if !ok {
		valid := make([]string, 0, len(generators))
		for k := range generators {
			valid = append(valid, k)
		}
		return nil, fmt.Errorf("unknown migration diff generator %q; available: %v", name, valid)
	}
	return g, nil
}

// DialectForAdapter returns the Ent dialect constant name for the given adapter.
func DialectForAdapter(adapter string) string {
	switch adapter {
	case "sqlite":
		return "SQLite"
	case "postgresql", "postgres":
		return "Postgres"
	case "mysql":
		return "MySQL"
	default:
		return "Postgres"
	}
}

// DriverImportForAdapter returns the Go driver import path for the given adapter.
func DriverImportForAdapter(adapter string) string {
	switch adapter {
	case "sqlite":
		return "github.com/mattn/go-sqlite3"
	case "postgresql", "postgres":
		return "github.com/lib/pq"
	case "mysql":
		return "github.com/go-sql-driver/mysql"
	default:
		return "github.com/lib/pq"
	}
}

// AtlasDialectForAdapter returns the Atlas dialect string for SQL parsing.
func AtlasDialectForAdapter(adapter string) string {
	switch adapter {
	case "sqlite":
		return "sqlite"
	case "postgresql", "postgres":
		return "postgres"
	case "mysql":
		return "mysql"
	default:
		return "postgres"
	}
}
