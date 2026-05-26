package migration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	atlassqlite "ariga.io/atlas/sql/sqlite"
	_ "github.com/mattn/go-sqlite3"

	"github.com/polagonow/pola/internal/generators/migration/diff"
)

// generateSchemaHCL replays all migrations on an in-memory dev DB and writes
// db/schema.hcl — a human-readable HCL snapshot of the full database
// schema (like Rails' db/schema.rb).
func generateSchemaHCL(ctx context.Context, cfg diff.Config) error {
	dir, err := migrate.NewLocalDir(cfg.MigrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	files, err := dir.Files()
	if err != nil {
		return fmt.Errorf("read migration files: %w", err)
	}
	if len(files) == 0 {
		return nil
	}

	// Open an in-memory SQLite database.
	db, err := sql.Open("sqlite3", "file:schema_hcl?mode=memory&cache=shared&_fk=1")
	if err != nil {
		return fmt.Errorf("open dev db: %w", err)
	}
	defer db.Close()

	drv, err := atlassqlite.Open(db)
	if err != nil {
		return fmt.Errorf("open atlas driver: %w", err)
	}

	// Replay all migrations using NopRevisionReadWriter.
	ex, err := migrate.NewExecutor(drv, dir, migrate.NopRevisionReadWriter{}, migrate.WithAllowDirty(true))
	if err != nil {
		return fmt.Errorf("create executor: %w", err)
	}

	if err := ex.ExecuteN(ctx, 0); err != nil {
		return fmt.Errorf("replay migrations: %w", err)
	}

	// Inspect the resulting schema.
	realm, err := drv.InspectRealm(ctx, &schema.InspectRealmOption{})
	if err != nil {
		return fmt.Errorf("inspect schema: %w", err)
	}

	// Marshal to HCL.
	hclBytes, err := atlassqlite.MarshalHCL(realm)
	if err != nil {
		return fmt.Errorf("marshal HCL: %w", err)
	}

	outputPath := filepath.Join(filepath.Dir(cfg.MigrationsDir), "schema.hcl")
	if err := os.WriteFile(outputPath, hclBytes, 0o644); err != nil {
		return fmt.Errorf("write schema.hcl: %w", err)
	}

	fmt.Printf("Updated %s\n", outputPath)
	return nil
}
