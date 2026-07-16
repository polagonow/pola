// Package migrator applies Atlas SQL migrations against a database. It is shared
// by the `pola db` CLI and by the embedded-migrations runtime plugin generated
// for `pola build --migrate`, so both drive migrations through identical logic
// and the same atlas_schema_revisions bookkeeping.
//
// The package deliberately does NOT import any database/sql driver. Callers are
// expected to have the appropriate driver registered already — the CLI blank-
// imports all three, and a built app registers its adapter's driver via the ORM
// adapter sub-package. This keeps the package CGO-neutral.
package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"ariga.io/atlas/sql/migrate"
	atlasmysql "ariga.io/atlas/sql/mysql"
	atlaspg "ariga.io/atlas/sql/postgres"
	atlassqlite "ariga.io/atlas/sql/sqlite"

	"github.com/polagonow/pola/database/dsn"
)

// OpenDriver opens a database connection for the given adapter and connection
// string and returns an Atlas migration driver alongside the raw *sql.DB.
// Supported adapters: "sqlite", "postgresql", "mysql". The corresponding
// database/sql driver must already be registered by the caller.
func OpenDriver(adapter, connStr string) (migrate.Driver, *sql.DB, error) {
	switch adapter {
	case "sqlite":
		// Strip any sqlite:// or sqlite: URL prefix; the driver wants a path.
		path := strings.TrimPrefix(connStr, "sqlite://")
		path = strings.TrimPrefix(path, "sqlite:")
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			return nil, nil, fmt.Errorf("open sqlite: %w", err)
		}
		drv, err := atlassqlite.Open(db)
		if err != nil {
			db.Close()
			return nil, nil, fmt.Errorf("open atlas sqlite driver: %w", err)
		}
		return drv, db, nil
	case "postgresql":
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			return nil, nil, fmt.Errorf("open postgres: %w", err)
		}
		drv, err := atlaspg.Open(db)
		if err != nil {
			db.Close()
			return nil, nil, fmt.Errorf("open atlas postgres driver: %w", err)
		}
		return drv, db, nil
	case "mysql":
		db, err := sql.Open("mysql", connStr)
		if err != nil {
			return nil, nil, fmt.Errorf("open mysql: %w", err)
		}
		drv, err := atlasmysql.Open(db)
		if err != nil {
			db.Close()
			return nil, nil, fmt.Errorf("open atlas mysql driver: %w", err)
		}
		return drv, db, nil
	default:
		return nil, nil, fmt.Errorf("adapter %q not supported for migrations; use sqlite, postgresql, or mysql", adapter)
	}
}

// RunPending applies every pending migration in dir against db. It is a no-op
// when the database is already up to date. Revision history is tracked in the
// adapter-appropriate atlas_schema_revisions table.
func RunPending(ctx context.Context, drv migrate.Driver, db *sql.DB, dir migrate.Dir, adapter string) (int, error) {
	rrw := NewRevisionReadWriter(db, adapter)
	if err := rrw.Init(ctx); err != nil {
		return 0, fmt.Errorf("init revision table: %w", err)
	}

	ex, err := migrate.NewExecutor(drv, dir, rrw, migrate.WithAllowDirty(true))
	if err != nil {
		return 0, fmt.Errorf("create executor: %w", err)
	}

	pending, err := ex.Pending(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "no pending") {
			return 0, nil
		}
		return 0, fmt.Errorf("check pending: %w", err)
	}
	if len(pending) == 0 {
		return 0, nil
	}
	if err := ex.ExecuteN(ctx, len(pending)); err != nil {
		return 0, fmt.Errorf("migrate: %w", err)
	}
	return len(pending), nil
}

// RunEmbedded applies all pending migrations from an embedded filesystem. The
// provided fs.FS must contain the migration files (.sql plus atlas.sum) at its
// root. cfg identifies the target database; empty fields fall back to DATABASE_*
// environment variables via the dsn package. The generated plugin passes the
// same config the app's ORM plugin resolves, so both connect to the same DB.
//
// This is the entry point used by the runtime plugin generated for
// `pola build --migrate`.
func RunEmbedded(ctx context.Context, cfg dsn.Config, migrationsFS fs.FS) (int, error) {
	c := cfg.WithEnvFallback()
	connStr := dsn.Build(c)

	drv, db, err := OpenDriver(c.Adapter, connStr)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	tmp, err := extractToTempDir(migrationsFS)
	if err != nil {
		return 0, fmt.Errorf("materialize embedded migrations: %w", err)
	}
	defer os.RemoveAll(tmp)

	dir, err := migrate.NewLocalDir(tmp)
	if err != nil {
		return 0, fmt.Errorf("open migrations dir: %w", err)
	}

	return RunPending(ctx, drv, db, dir, c.Adapter)
}

// extractToTempDir copies every file in fsys into a fresh temp directory,
// preserving relative paths, so the on-disk migrate.NewLocalDir path (including
// atlas.sum integrity checks) can be reused verbatim against embedded files.
func extractToTempDir(fsys fs.FS) (string, error) {
	tmp, err := os.MkdirTemp("", "pola-migrations-*")
	if err != nil {
		return "", err
	}
	walkErr := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == "." {
				return nil
			}
			return os.MkdirAll(filepath.Join(tmp, path), 0o755)
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(tmp, path), data, 0o644)
	})
	if walkErr != nil {
		os.RemoveAll(tmp)
		return "", walkErr
	}
	return tmp, nil
}
