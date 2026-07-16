package dbembed

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polagonow/pola/internal/autoload"
)

// newTestProject creates a temp project with a migrations directory. When
// withSchema is true it also writes db/schema.hcl.
func newTestProject(t *testing.T, withMigration, withSchema bool) string {
	t.Helper()
	dir := t.TempDir()
	migDir := filepath.Join(dir, "db", "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if withMigration {
		if err := os.WriteFile(filepath.Join(migDir, "20260101000000_init.sql"), []byte("CREATE TABLE t (id integer);\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(migDir, "atlas.sum"), []byte("h1:test\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if withSchema {
		if err := os.WriteFile(filepath.Join(dir, "db", "schema.hcl"), []byte("schema \"main\" {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func runContribute(t *testing.T, projectDir string, opts autoload.PluginOpts) (*autoload.Context, string) {
	t.Helper()
	ctx := &autoload.Context{
		ProjectDir: projectDir,
		TmpDir:     t.TempDir(),
		Opts:       opts,
		Replace:    map[string]string{},
		Discovery:  &autoload.Discovery{},
	}
	if err := (&autoloadImpl{}).Contribute(ctx); err != nil {
		t.Fatalf("Contribute: %v", err)
	}
	absProject, _ := filepath.Abs(projectDir)
	realPath, ok := ctx.Replace[filepath.Join(absProject, "pola_migrate.go")]
	if !ok {
		return ctx, ""
	}
	data, err := os.ReadFile(realPath)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	return ctx, string(data)
}

// TestContribute_GeneratesValidPluginWithBakedConfig verifies that the generated
// pola_migrate.go is valid Go, embeds migrations + schema, and bakes the same
// connection config (adapter, name) that the ORM plugin uses.
func TestContribute_GeneratesValidPluginWithBakedConfig(t *testing.T) {
	projectDir := newTestProject(t, true, true)
	opts := autoload.PluginOpts{
		PolaPackage:           "github.com/polagonow/pola",
		EmbedMigrations:       true,
		Migrate:               true,
		Database:              "gorm",
		DatabaseAdapter:       "postgresql",
		DatabaseName:          "app.db",
		DatabaseMigrationsDir: "db/migrations",
	}
	ctx, src := runContribute(t, projectDir, opts)
	if src == "" {
		t.Fatal("expected pola_migrate.go to be generated")
	}
	if !ctx.Discovery.EmbeddedMigrations {
		t.Error("expected Discovery.EmbeddedMigrations to be set")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "pola_migrate.go", src, parser.AllErrors); err != nil {
		t.Fatalf("generated source does not parse: %v\n%s", err, src)
	}
	for _, want := range []string{
		"//go:embed all:db/migrations",
		"//go:embed db/schema.hcl",
		"func migratePlugin() core.Plugin",
		`polaenv.String("POLA_DATABASE_ADAPTER", "postgresql")`,
		`polaenv.String("POLA_DATABASE_NAME", "app.db")`,
		"migrator.RunEmbedded(ctx, cfg, sub)",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated source missing %q\n%s", want, src)
		}
	}
	// Fields not configured in the Polafile must not be baked.
	if strings.Contains(src, "POLA_DATABASE_HOST") {
		t.Errorf("did not expect POLA_DATABASE_HOST baking when host unset\n%s", src)
	}
}

// TestContribute_NoSchemaOmitsSchemaEmbed verifies schema embedding is skipped
// when db/schema.hcl is absent.
func TestContribute_NoSchemaOmitsSchemaEmbed(t *testing.T) {
	projectDir := newTestProject(t, true, false)
	opts := autoload.PluginOpts{
		PolaPackage:     "github.com/polagonow/pola",
		EmbedMigrations: true,
		Database:        "gorm",
		DatabaseAdapter: "sqlite",
	}
	_, src := runContribute(t, projectDir, opts)
	if src == "" {
		t.Fatal("expected pola_migrate.go to be generated")
	}
	if strings.Contains(src, "schema.hcl") {
		t.Errorf("did not expect schema embed when db/schema.hcl absent\n%s", src)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "pola_migrate.go", src, parser.AllErrors); err != nil {
		t.Fatalf("generated source does not parse: %v\n%s", err, src)
	}
}

// TestContribute_SkipsWhenNoMigrations verifies that no overlay is produced (and
// the Discovery flag stays false) when there are no .sql files to embed.
func TestContribute_SkipsWhenNoMigrations(t *testing.T) {
	projectDir := newTestProject(t, false, true)
	opts := autoload.PluginOpts{
		PolaPackage:     "github.com/polagonow/pola",
		EmbedMigrations: true,
		Migrate:         true,
		Database:        "gorm",
		DatabaseAdapter: "sqlite",
	}
	ctx, src := runContribute(t, projectDir, opts)
	if src != "" {
		t.Errorf("expected no overlay when there are no migrations\n%s", src)
	}
	if ctx.Discovery.EmbeddedMigrations {
		t.Error("expected Discovery.EmbeddedMigrations to stay false when skipped")
	}
}

// TestContribute_DisabledIsNoOp verifies the autoload does nothing when embedding
// was not requested or no database is configured.
func TestContribute_DisabledIsNoOp(t *testing.T) {
	projectDir := newTestProject(t, true, true)

	// Not requested.
	ctx, src := runContribute(t, projectDir, autoload.PluginOpts{
		PolaPackage: "github.com/polagonow/pola",
		Database:    "gorm",
	})
	if src != "" || ctx.Discovery.EmbeddedMigrations {
		t.Error("expected no-op when EmbedMigrations is false")
	}

	// No database configured.
	ctx, src = runContribute(t, projectDir, autoload.PluginOpts{
		PolaPackage:     "github.com/polagonow/pola",
		EmbedMigrations: true,
	})
	if src != "" || ctx.Discovery.EmbeddedMigrations {
		t.Error("expected no-op when no database is configured")
	}
}
