package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	atlassqlite "ariga.io/atlas/sql/sqlite"
	_ "github.com/mattn/go-sqlite3"

	"github.com/polagonow/pola/database/dsn"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
	Long:  "Run database migrations, check status, rollback, and more.",
}

func init() {
	dbCmd.AddCommand(dbMigrateCmd)
	dbCmd.AddCommand(dbRollbackCmd)
	dbCmd.AddCommand(dbStatusCmd)
	dbCmd.AddCommand(dbResetCmd)
	dbCmd.AddCommand(dbSchemaLoadCmd)
}

// ---------- migrate ----------

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run pending migrations",
	Long: `Apply all pending migrations to the database.
Use --version to migrate to a specific version.`,
	RunE: runDBMigrate,
	Example: `  pola db migrate
  pola db migrate --url "sqlite:dev.db"
  pola db migrate --version 20240101120000`,
}

func init() {
	dbMigrateCmd.Flags().String("url", "", "Database URL (overrides Polafile)")
	dbMigrateCmd.Flags().String("env", "development", "Environment")
	dbMigrateCmd.Flags().String("version", "", "Migrate to specific version")
}

func runDBMigrate(cmd *cobra.Command, _ []string) error {
	cfg, err := loadDBConfig(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	drv, db, err := openDriver(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	dir, err := migrate.NewLocalDir(cfg.migrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	rrw := newRevisionReadWriter(db, cfg.adapter)
	if err := rrw.init(ctx); err != nil {
		return fmt.Errorf("init revision table: %w", err)
	}

	ex, err := migrate.NewExecutor(drv, dir, rrw, migrate.WithAllowDirty(true))
	if err != nil {
		return fmt.Errorf("create executor: %w", err)
	}

	version, _ := cmd.Flags().GetString("version")
	if version != "" {
		if err := ex.ExecuteTo(ctx, version); err != nil {
			return fmt.Errorf("migrate to version %s: %w", version, err)
		}
		fmt.Printf("Migrated to version %s\n", version)
	} else {
		pending, err := ex.Pending(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "no pending") {
				fmt.Println("Database is up to date.")
				return nil
			}
			return fmt.Errorf("check pending: %w", err)
		}
		if len(pending) == 0 {
			fmt.Println("Database is up to date.")
			return nil
		}
		if err := ex.ExecuteN(ctx, len(pending)); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		for _, f := range pending {
			fmt.Printf("  %s  %s\n", f.Version(), f.Desc())
		}
		fmt.Printf("Applied %d migration(s)\n", len(pending))
	}

	return nil
}

// ---------- rollback ----------

var dbRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback the last migration",
	Long: `Rollback the last applied migration.
Use --step to rollback multiple migrations.`,
	RunE: runDBRollback,
	Example: `  pola db rollback
  pola db rollback --step 3
  pola db rollback --url "sqlite:dev.db"`,
}

func init() {
	dbRollbackCmd.Flags().String("url", "", "Database URL (overrides Polafile)")
	dbRollbackCmd.Flags().String("env", "development", "Environment")
	dbRollbackCmd.Flags().Int("step", 1, "Number of migrations to rollback")
}

func runDBRollback(cmd *cobra.Command, _ []string) error {
	cfg, err := loadDBConfig(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	_, db, err := openDriver(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	rrw := newRevisionReadWriter(db, cfg.adapter)
	if err := rrw.init(ctx); err != nil {
		return fmt.Errorf("init revision table: %w", err)
	}

	step, _ := cmd.Flags().GetInt("step")
	revisions, err := rrw.ReadRevisions(ctx)
	if err != nil {
		return fmt.Errorf("read revisions: %w", err)
	}
	if len(revisions) == 0 {
		fmt.Println("No migrations to rollback.")
		return nil
	}

	// Get the last N revisions to rollback.
	count := step
	if count > len(revisions) {
		count = len(revisions)
	}

	dir, err := migrate.NewLocalDir(cfg.migrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	files, err := dir.Files()
	if err != nil {
		return fmt.Errorf("read migration files: %w", err)
	}

	// Build a map of version -> file for reverse statement lookup.
	fileMap := make(map[string]migrate.File)
	for _, f := range files {
		fileMap[f.Version()] = f
	}

	// Rollback in reverse order.
	rolled := 0
	for i := len(revisions) - 1; i >= 0 && rolled < count; i-- {
		rev := revisions[i]
		f, ok := fileMap[rev.Version]
		if !ok {
			return fmt.Errorf("migration file for version %s not found", rev.Version)
		}

		// Try to extract reverse statements from the migration file.
		stmts, err := f.StmtDecls()
		if err != nil {
			return fmt.Errorf("parse migration %s: %w", rev.Version, err)
		}

		var reverseSQL []string
		for _, s := range stmts {
			for _, d := range s.Directive("atlas:down") {
				reverseSQL = append(reverseSQL, d)
			}
		}

		if len(reverseSQL) == 0 {
			return fmt.Errorf("migration %s (%s) has no reverse statements (add -- atlas:down directives)", rev.Version, rev.Description)
		}

		fmt.Printf("Rolling back %s (%s)...\n", rev.Version, rev.Description)
		for _, stmt := range reverseSQL {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("rollback %s: execute %q: %w", rev.Version, stmt, err)
			}
		}

		if err := rrw.DeleteRevision(ctx, rev.Version); err != nil {
			return fmt.Errorf("delete revision %s: %w", rev.Version, err)
		}

		rolled++
	}

	fmt.Printf("Rolled back %d migration(s)\n", rolled)
	return nil
}

// ---------- status ----------

var dbStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long:  "Show which migrations have been applied and which are pending.",
	RunE:  runDBStatus,
	Example: `  pola db status
  pola db status --url "sqlite:dev.db"`,
}

func init() {
	dbStatusCmd.Flags().String("url", "", "Database URL (overrides Polafile)")
	dbStatusCmd.Flags().String("env", "development", "Environment")
}

func runDBStatus(cmd *cobra.Command, _ []string) error {
	cfg, err := loadDBConfig(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	drv, db, err := openDriver(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	dir, err := migrate.NewLocalDir(cfg.migrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	rrw := newRevisionReadWriter(db, cfg.adapter)
	if err := rrw.init(ctx); err != nil {
		return fmt.Errorf("init revision table: %w", err)
	}

	ex, err := migrate.NewExecutor(drv, dir, rrw, migrate.WithAllowDirty(true))
	if err != nil {
		return fmt.Errorf("create executor: %w", err)
	}

	revisions, err := rrw.ReadRevisions(ctx)
	if err != nil {
		return fmt.Errorf("read revisions: %w", err)
	}

	pending, err := ex.Pending(ctx)
	if err != nil && !strings.Contains(err.Error(), "no pending") {
		return fmt.Errorf("check pending: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "VERSION\tDESCRIPTION\tSTATUS\tAPPLIED AT")
	fmt.Fprintln(w, "-------\t-----------\t------\t----------")

	appliedMap := make(map[string]*migrate.Revision)
	for _, r := range revisions {
		appliedMap[r.Version] = r
	}

	// Show applied migrations.
	for _, r := range revisions {
		status := "applied"
		if r.Error != "" {
			status = "error: " + r.Error
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Version, r.Description, status, r.ExecutedAt.Format(time.RFC3339))
	}

	// Show pending migrations.
	for _, f := range pending {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", f.Version(), f.Desc(), "pending", "-")
	}

	w.Flush()

	fmt.Printf("\nApplied: %d, Pending: %d\n", len(revisions), len(pending))
	return nil
}

// ---------- reset ----------

var dbResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Drop all tables and re-run migrations",
	Long:  "Drop all tables in the database, then re-run all migrations from scratch.",
	RunE:  runDBReset,
	Example: `  pola db reset
  pola db reset --url "sqlite:dev.db"`,
}

func init() {
	dbResetCmd.Flags().String("url", "", "Database URL (overrides Polafile)")
	dbResetCmd.Flags().String("env", "development", "Environment")
}

func runDBReset(cmd *cobra.Command, _ []string) error {
	cfg, err := loadDBConfig(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	drv, db, err := openDriver(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	// Inspect current schema and drop all tables.
	fmt.Println("Dropping all tables...")
	realm, err := drv.InspectRealm(ctx, &schema.InspectRealmOption{})
	if err != nil {
		return fmt.Errorf("inspect schema: %w", err)
	}

	for _, s := range realm.Schemas {
		for _, t := range s.Tables {
			stmt := fmt.Sprintf("DROP TABLE IF EXISTS %q", t.Name)
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("drop table %s: %w", t.Name, err)
			}
		}
	}

	// Re-run all migrations.
	fmt.Println("Re-running all migrations...")
	dir, err := migrate.NewLocalDir(cfg.migrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	rrw := newRevisionReadWriter(db, cfg.adapter)
	// Recreate the revision table.
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS atlas_schema_revisions"); err != nil {
		return fmt.Errorf("drop revision table: %w", err)
	}
	if err := rrw.init(ctx); err != nil {
		return fmt.Errorf("init revision table: %w", err)
	}

	ex, err := migrate.NewExecutor(drv, dir, rrw, migrate.WithAllowDirty(true))
	if err != nil {
		return fmt.Errorf("create executor: %w", err)
	}

	pending, err := ex.Pending(ctx)
	if err != nil {
		return fmt.Errorf("check pending: %w", err)
	}

	if err := ex.ExecuteN(ctx, len(pending)); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	fmt.Printf("Applied %d migration(s)\n", len(pending))
	return nil
}

// ---------- schema:load ----------

var dbSchemaLoadCmd = &cobra.Command{
	Use:   "schema:load",
	Short: "Replay all migrations on a fresh database",
	Long:  "Apply all migration files to a fresh database, useful for setting up a new environment.",
	RunE:  runDBSchemaLoad,
	Example: `  pola db schema:load
  pola db schema:load --url "sqlite:dev.db"`,
}

func init() {
	dbSchemaLoadCmd.Flags().String("url", "", "Database URL (overrides Polafile)")
	dbSchemaLoadCmd.Flags().String("env", "development", "Environment")
}

func runDBSchemaLoad(cmd *cobra.Command, _ []string) error {
	cfg, err := loadDBConfig(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	drv, db, err := openDriver(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	dir, err := migrate.NewLocalDir(cfg.migrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	rrw := newRevisionReadWriter(db, cfg.adapter)
	if err := rrw.init(ctx); err != nil {
		return fmt.Errorf("init revision table: %w", err)
	}

	ex, err := migrate.NewExecutor(drv, dir, rrw, migrate.WithAllowDirty(true))
	if err != nil {
		return fmt.Errorf("create executor: %w", err)
	}

	pending, err := ex.Pending(ctx)
	if err != nil {
		return fmt.Errorf("check pending: %w", err)
	}

	if err := ex.ExecuteN(ctx, len(pending)); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	fmt.Printf("Applied %d migration(s)\n", len(pending))
	return nil
}

// ---------- shared helpers ----------

type dbConfig struct {
	projectDir    string
	migrationsDir string
	url           string
	adapter       string
}

func loadDBConfig(cmd *cobra.Command) (*dbConfig, error) {
	projectDir, err := project.FindRoot()
	if err != nil {
		return nil, err
	}

	pf, err := polafile.Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		return nil, fmt.Errorf("no Polafile.hcl found; run 'pola new' first")
	}

	env, _ := cmd.Flags().GetString("env")
	urlFlag, _ := cmd.Flags().GetString("url")

	url := urlFlag
	if url == "" {
		url = pf.DatabaseURL(env)
	}
	// If no explicit URL, try building from individual fields.
	if url == "" {
		merged := pf.DatabaseForEnv(env)
		if merged.Host != "" || merged.Name != "" {
			url = buildDSN(merged)
		}
	}
	// If still no URL, derive a sensible default from the adapter (with env var fallback).
	if url == "" {
		adapter := pf.DatabaseAdapter(env)
		url = dsn.Build(dsn.Config{Adapter: adapter}.WithEnvFallback())
	}

	adapter := pf.DatabaseAdapter(env)
	migrationsDir := filepath.Join(projectDir, pf.DatabaseMigrationsDir())
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("migrations directory %q does not exist; run 'pola generate migration' first", migrationsDir)
	}

	return &dbConfig{
		projectDir:    projectDir,
		migrationsDir: migrationsDir,
		url:           url,
		adapter:       adapter,
	}, nil
}

// buildDSN constructs a connection string from individual Polafile database fields.
func buildDSN(db polafile.Database) string {
	return dsn.Build(dsn.Config{
		URL:      db.URL,
		Host:     db.Host,
		Port:     db.Port,
		User:     db.User,
		Password: db.Password,
		Name:     db.Name,
		Adapter:  db.Adapter,
	})
}

// openDriver opens a database connection and returns an Atlas driver.
// Currently supports sqlite. Postgres/MySQL can be added with their respective drivers.
func openDriver(cfg *dbConfig) (migrate.Driver, *sql.DB, error) {
	switch cfg.adapter {
	case "sqlite":
		dsn := cfg.url
		// Strip sqlite:// prefix if present.
		dsn = strings.TrimPrefix(dsn, "sqlite://")
		dsn = strings.TrimPrefix(dsn, "sqlite:")
		db, err := sql.Open("sqlite3", dsn)
		if err != nil {
			return nil, nil, fmt.Errorf("open sqlite: %w", err)
		}
		drv, err := atlassqlite.Open(db)
		if err != nil {
			db.Close()
			return nil, nil, fmt.Errorf("open atlas sqlite driver: %w", err)
		}
		return drv, db, nil
	default:
		return nil, nil, fmt.Errorf("adapter %q not yet supported for pola db commands; currently only sqlite is supported", cfg.adapter)
	}
}

// ---------- revision tracking ----------

// sqlRevisionReadWriter is a SQL-based RevisionReadWriter that stores migration
// history in an atlas_schema_revisions table.
type sqlRevisionReadWriter struct {
	db      *sql.DB
	adapter string
}

func newRevisionReadWriter(db *sql.DB, adapter string) *sqlRevisionReadWriter {
	return &sqlRevisionReadWriter{db: db, adapter: adapter}
}

func (rw *sqlRevisionReadWriter) init(ctx context.Context) error {
	_, err := rw.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS atlas_schema_revisions (
		version TEXT PRIMARY KEY,
		description TEXT NOT NULL DEFAULT '',
		type INTEGER NOT NULL DEFAULT 0,
		applied INTEGER NOT NULL DEFAULT 0,
		total INTEGER NOT NULL DEFAULT 0,
		executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		execution_time INTEGER NOT NULL DEFAULT 0,
		error TEXT NOT NULL DEFAULT '',
		error_stmt TEXT NOT NULL DEFAULT '',
		hash TEXT NOT NULL DEFAULT '',
		operator_version TEXT NOT NULL DEFAULT ''
	)`)
	return err
}

func (rw *sqlRevisionReadWriter) Ident() *migrate.TableIdent {
	return &migrate.TableIdent{Name: "atlas_schema_revisions"}
}

func (rw *sqlRevisionReadWriter) ReadRevisions(ctx context.Context) ([]*migrate.Revision, error) {
	rows, err := rw.db.QueryContext(ctx, `SELECT version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, operator_version FROM atlas_schema_revisions ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revs []*migrate.Revision
	for rows.Next() {
		var r migrate.Revision
		var execTime int64
		if err := rows.Scan(&r.Version, &r.Description, &r.Type, &r.Applied, &r.Total, &r.ExecutedAt, &execTime, &r.Error, &r.ErrorStmt, &r.Hash, &r.OperatorVersion); err != nil {
			return nil, err
		}
		r.ExecutionTime = time.Duration(execTime)
		revs = append(revs, &r)
	}
	return revs, rows.Err()
}

func (rw *sqlRevisionReadWriter) ReadRevision(ctx context.Context, version string) (*migrate.Revision, error) {
	var r migrate.Revision
	var execTime int64
	err := rw.db.QueryRowContext(ctx, `SELECT version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, operator_version FROM atlas_schema_revisions WHERE version = ?`, version).
		Scan(&r.Version, &r.Description, &r.Type, &r.Applied, &r.Total, &r.ExecutedAt, &execTime, &r.Error, &r.ErrorStmt, &r.Hash, &r.OperatorVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, migrate.ErrRevisionNotExist
	}
	if err != nil {
		return nil, err
	}
	r.ExecutionTime = time.Duration(execTime)
	return &r, nil
}

func (rw *sqlRevisionReadWriter) WriteRevision(ctx context.Context, r *migrate.Revision) error {
	_, err := rw.db.ExecContext(ctx, `INSERT OR REPLACE INTO atlas_schema_revisions (version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, operator_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Version, r.Description, r.Type, r.Applied, r.Total, r.ExecutedAt, int64(r.ExecutionTime), r.Error, r.ErrorStmt, r.Hash, r.OperatorVersion)
	return err
}

func (rw *sqlRevisionReadWriter) DeleteRevision(ctx context.Context, version string) error {
	_, err := rw.db.ExecContext(ctx, `DELETE FROM atlas_schema_revisions WHERE version = ?`, version)
	return err
}
