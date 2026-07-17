package cli

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/polagonow/pola/database/dsn"
	"github.com/polagonow/pola/database/migrator"
	"github.com/polagonow/pola/internal/autoload"
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
	dbCmd.AddCommand(dbSeedCmd)
}

// ---------- seed ----------

var dbSeedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Run database seeds (db/seeds)",
	Long: `Build the app and run the seeders registered in db/seeds.

Define db/seeds/seeds.go with:

    func Seed(ctx context.Context, r *core.Registry) error

(scaffold it via 'pola generate seed'). The seeder runs with the full DI
registry, so it can resolve services and repositories via core.MustInvoke.`,
	RunE:    runDBSeed,
	Example: `  pola db seed`,
}

func runDBSeed(cmd *cobra.Command, _ []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	var pf polafile.Polafile
	pfPtr, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	pfPtr = polafile.ApplyEnvOverrides(pfPtr)
	if pfPtr != nil {
		pf = *pfPtr
	}
	isAPIOnly := pfPtr != nil && pfPtr.IsAPIOnly()

	// Generate the plugin/route/seed overlay, exactly as the dev server does, so
	// the app compiles with db/seeds wired in.
	opts := seedPluginOpts(&pf, isAPIOnly)
	overlayRes, err := autoload.Run(projectDir, opts, verbose)
	if err != nil {
		return err
	}
	defer cleanupOverlay(overlayRes)

	goArgs := []string{"run"}
	if overlayRes != nil && overlayRes.OverlayPath != "" {
		goArgs = append(goArgs, "-overlay", overlayRes.OverlayPath)
	}
	goArgs = append(goArgs, ".")

	run := exec.Command("go", goArgs...)
	run.Dir = projectDir
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	// POLA_SEED_ONLY makes pola.Ready() run the seeders and exit instead of serving.
	run.Env = append(os.Environ(),
		"POLA_ENV=development",
		"POLA_SEED_ONLY=true",
		"GOFLAGS=-mod=mod",
	)
	return run.Run()
}

// seedPluginOpts builds the overlay PluginOpts from the Polafile (mirroring the
// dev-server wiring) so `pola db seed` compiles the app with all plugins.
func seedPluginOpts(pf *polafile.Polafile, isAPIOnly bool) autoload.PluginOpts {
	opts := autoload.PluginOpts{
		PolaPackage:     pf.PolaPackage(),
		Framework:       cmp.Or(os.Getenv("POLA_WEB_FRAMEWORK"), pf.Framework, "std"),
		Cache:           cmp.Or(os.Getenv("POLA_CACHE"), pf.CacheAdapter("default"), "memory"),
		CSRF:            autoload.EnvOrBool("POLA_CSRF", pf.CSRFEnabled("default")),
		SecurityHeaders: autoload.EnvOrBool("POLA_SECURITY_HEADERS", pf.SecurityHeadersEnabled("default")),
		Dev:             true,
		APIOnly:         isAPIOnly,
	}
	if !isAPIOnly {
		opts.Engine = cmp.Or(os.Getenv("POLA_VM"), nameOnly(pf.Engine), "goja")
		opts.Bundler = cmp.Or(os.Getenv("POLA_BUNDLER"), nameOnly(pf.Bundler), "esbuild")
		opts.Renderer = cmp.Or(os.Getenv("POLA_RENDERER"), nameOnly(pf.Renderer), "react")
		opts.Router = cmp.Or(os.Getenv("POLA_ROUTER"), nameOnly(pf.Router), "nextjs")
		opts.CSS = cmp.Or(os.Getenv("POLA_CSS"), nameOnly(pf.CSS), "tailwind")
		opts.AppDir = pf.AppDir()
	}
	defaultEnv := "development"
	autoload.PopulateDatabaseOpts(&opts, pf, defaultEnv)
	autoload.PopulateStorageOpts(&opts, pf, defaultEnv)
	autoload.ApplyMailerOpts(&opts, pf, defaultEnv)
	autoload.PopulateMCPOpts(&opts, pf, defaultEnv)
	autoload.PopulateSessionOpts(&opts, pf, defaultEnv)
	autoload.PopulateJWTOpts(&opts, pf, defaultEnv)
	autoload.PopulateProtectOpts(&opts, pf, defaultEnv)
	autoload.PopulateCSRFOpts(&opts, pf, defaultEnv)
	autoload.PopulateRateLimitOpts(&opts, pf, defaultEnv)
	autoload.PopulateFlashOpts(&opts, pf, defaultEnv)
	autoload.PopulateI18nOpts(&opts, pf, defaultEnv)
	return opts
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

	drv, db, err := migrator.OpenDriver(cfg.adapter, cfg.url)
	if err != nil {
		return err
	}
	defer db.Close()

	dir, err := migrate.NewLocalDir(cfg.migrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	rrw := migrator.NewRevisionReadWriter(db, cfg.adapter)
	if err := rrw.Init(ctx); err != nil {
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

	_, db, err := migrator.OpenDriver(cfg.adapter, cfg.url)
	if err != nil {
		return err
	}
	defer db.Close()

	rrw := migrator.NewRevisionReadWriter(db, cfg.adapter)
	if err := rrw.Init(ctx); err != nil {
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

	drv, db, err := migrator.OpenDriver(cfg.adapter, cfg.url)
	if err != nil {
		return err
	}
	defer db.Close()

	dir, err := migrate.NewLocalDir(cfg.migrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	rrw := migrator.NewRevisionReadWriter(db, cfg.adapter)
	if err := rrw.Init(ctx); err != nil {
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

	drv, db, err := migrator.OpenDriver(cfg.adapter, cfg.url)
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

	rrw := migrator.NewRevisionReadWriter(db, cfg.adapter)
	// Recreate the revision table.
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS atlas_schema_revisions"); err != nil {
		return fmt.Errorf("drop revision table: %w", err)
	}
	if err := rrw.Init(ctx); err != nil {
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

	drv, db, err := migrator.OpenDriver(cfg.adapter, cfg.url)
	if err != nil {
		return err
	}
	defer db.Close()

	dir, err := migrate.NewLocalDir(cfg.migrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	rrw := migrator.NewRevisionReadWriter(db, cfg.adapter)
	if err := rrw.Init(ctx); err != nil {
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
