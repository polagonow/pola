// Package dbembed implements the database-migration embedding overlay autoload.
// When `pola build --embed-migrations` (or --migrate) is used, it generates a
// pola_migrate.go overlay that embeds the project's db/migrations directory (and
// db/schema.hcl) into the binary and, for --migrate, exposes a migratePlugin()
// that runs the embedded migrations on boot.
package dbembed

import (
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/autoload"
)

//go:embed _templates/pola_migrate.go.tmpl
var templates embed.FS

var migrateTmpl = template.Must(
	template.New("pola_migrate.go.tmpl").ParseFS(templates, "_templates/pola_migrate.go.tmpl"),
)

type autoloadImpl struct{}

// New returns this autoload stage for explicit registration in autoload/all.
func New() autoload.Autoload { return &autoloadImpl{} }

func (a *autoloadImpl) Name() string { return "dbembed" }

// Priority runs before pluginimports (900) so that Discovery.EmbeddedMigrations
// is set before the plugin manifest decides whether to reference migratePlugin().
func (a *autoloadImpl) Priority() int { return 850 }

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	// Only when embedding was requested and a database ORM is configured.
	if !ctx.Opts.EmbedMigrations || ctx.Opts.Database == "" {
		return nil
	}

	migrationsDir := filepath.ToSlash(filepath.Clean(ctx.Opts.DatabaseMigrationsDir))
	if migrationsDir == "" || migrationsDir == "." {
		migrationsDir = "db/migrations"
	}

	absProjectDir, err := filepath.Abs(ctx.ProjectDir)
	if err != nil {
		return fmt.Errorf("abs project dir: %w", err)
	}

	// //go:embed fails to compile against a missing or empty directory, so skip
	// generation (with a warning) when there are no migrations to embed.
	absMigrations := filepath.Join(absProjectDir, filepath.FromSlash(migrationsDir))
	if !dirHasMigrations(absMigrations) {
		fmt.Printf("Warning: --embed-migrations set but no .sql migrations found in %q; skipping embed\n", migrationsDir)
		return nil
	}

	// schema.hcl lives alongside the migrations directory (e.g. db/schema.hcl).
	schemaPath := path.Join(path.Dir(migrationsDir), "schema.hcl")
	hasSchema := fileExists(filepath.Join(absProjectDir, filepath.FromSlash(schemaPath)))

	data := struct {
		PolaPackage   string
		MigrationsDir string
		Adapter       string
		DatabaseURL   string
		DatabaseHost  string
		DatabasePort  string
		DatabaseUser  string
		DatabasePass  string
		DatabaseName  string
		HasSchema     bool
		SchemaPath    string
	}{
		PolaPackage:   ctx.Opts.PolaPackage,
		MigrationsDir: migrationsDir,
		Adapter:       ctx.Opts.DatabaseAdapter,
		DatabaseURL:   ctx.Opts.DatabaseURL,
		DatabaseHost:  ctx.Opts.DatabaseHost,
		DatabasePort:  ctx.Opts.DatabasePort,
		DatabaseUser:  ctx.Opts.DatabaseUser,
		DatabasePass:  ctx.Opts.DatabasePass,
		DatabaseName:  ctx.Opts.DatabaseName,
		HasSchema:     hasSchema,
		SchemaPath:    schemaPath,
	}

	var buf strings.Builder
	if err := migrateTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute migrate template: %w", err)
	}

	migratePath := filepath.Join(ctx.TmpDir, "pola_migrate.go")
	if err := os.WriteFile(migratePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write pola_migrate.go: %w", err)
	}
	ctx.Replace[filepath.Join(absProjectDir, "pola_migrate.go")] = migratePath

	// Signal pluginimports that migratePlugin() now exists and may be wired.
	if ctx.Discovery != nil {
		ctx.Discovery.EmbeddedMigrations = true
	}

	return nil
}

// dirHasMigrations reports whether dir contains at least one .sql migration file.
func dirHasMigrations(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			return true
		}
	}
	return false
}

// fileExists reports whether path exists and is a regular file.
func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
