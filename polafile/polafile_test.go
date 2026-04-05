package polafile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseVersioned(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"tailwind@^4.0.0", "tailwind", "^4.0.0"},
		{"react@^19.0.0", "react", "^19.0.0"},
		{"pnpm@9.15.0", "pnpm", "9.15.0"},
		{"goja", "goja", ""},
		{"esbuild@*", "esbuild", "*"},
	}
	for _, tt := range tests {
		name, ver := ParseVersioned(tt.input)
		if name != tt.wantName || ver != tt.wantVersion {
			t.Errorf("ParseVersioned(%q) = (%q, %q), want (%q, %q)",
				tt.input, name, ver, tt.wantName, tt.wantVersion)
		}
	}
}

func TestFormatVersioned(t *testing.T) {
	if got := FormatVersioned("tailwind", "^4.0.0"); got != "tailwind@^4.0.0" {
		t.Errorf("FormatVersioned = %q, want %q", got, "tailwind@^4.0.0")
	}
	if got := FormatVersioned("goja", ""); got != "goja" {
		t.Errorf("FormatVersioned = %q, want %q", got, "goja")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	pf := &Polafile{
		Version:         "0.1.0",
		Renderer:        "react@^19.0.0",
		Engine:          "goja@0.0.0-20240220",
		Bundler:         "esbuild@^0.21.0",
		Router:          "nextjs",
		CSS:             "tailwind@^4.0.0",
		PackageManager:  "pnpm@^9.0.0",
		Cache:           &Cache{Enabled: true, Adapter: "memory"},
		CSRF:            &CSRF{Enabled: true},
		SecurityHeaders: &SecurityHeaders{Enabled: true},
		App:             "app",
		Actions:         "actions",
		Routes:          "routes",
	}

	if err := Save(dir, pf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists.
	path := filepath.Join(dir, Filename)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Polafile.hcl not created: %v", err)
	}

	// Load it back.
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("Load returned nil")
	}

	if loaded.Version != "0.1.0" {
		t.Errorf("Version = %q, want %q", loaded.Version, "0.1.0")
	}
	if loaded.Renderer != "react@^19.0.0" {
		t.Errorf("Renderer = %q, want %q", loaded.Renderer, "react@^19.0.0")
	}
	if loaded.Engine != "goja@0.0.0-20240220" {
		t.Errorf("Engine = %q, want %q", loaded.Engine, "goja@0.0.0-20240220")
	}
	if loaded.Bundler != "esbuild@^0.21.0" {
		t.Errorf("Bundler = %q, want %q", loaded.Bundler, "esbuild@^0.21.0")
	}
	if loaded.CSS != "tailwind@^4.0.0" {
		t.Errorf("CSS = %q, want %q", loaded.CSS, "tailwind@^4.0.0")
	}
	if loaded.CacheAdapter("default") != "memory" {
		t.Errorf("CacheAdapter(default) = %q, want %q", loaded.CacheAdapter("default"), "memory")
	}
	if loaded.PackageManager != "pnpm@^9.0.0" {
		t.Errorf("PackageManager = %q, want %q", loaded.PackageManager, "pnpm@^9.0.0")
	}
	if loaded.App != "app" {
		t.Errorf("App = %q, want %q", loaded.App, "app")
	}
	if loaded.Actions != "actions" {
		t.Errorf("Actions = %q, want %q", loaded.Actions, "actions")
	}
	if loaded.Routes != "routes" {
		t.Errorf("Routes = %q, want %q", loaded.Routes, "routes")
	}
	if !loaded.CSRFEnabled("default") {
		t.Error("CSRFEnabled(default) = false, want true")
	}
	if !loaded.SecurityHeadersEnabled("default") {
		t.Error("SecurityHeadersEnabled(default) = false, want true")
	}

	// Verify ParseVersioned works with loaded values.
	name, ver := ParseVersioned(loaded.CSS)
	if name != "tailwind" || ver != "^4.0.0" {
		t.Errorf("ParseVersioned(CSS) = (%q, %q), want (\"tailwind\", \"^4.0.0\")", name, ver)
	}
}

func TestSaveAndLoadWithCacheBlocks(t *testing.T) {
	dir := t.TempDir()

	pf := &Polafile{
		Renderer: "react",
		Cache: &Cache{
			Adapter: "memory",
			Envs: []CacheEnvironment{
				{Environment: "production", Adapter: "redis", Host: "localhost", Port: "6379"},
			},
		},
	}

	if err := Save(dir, pf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.CacheAdapter("default") != "memory" {
		t.Errorf("CacheAdapter(default) = %q, want %q", loaded.CacheAdapter("default"), "memory")
	}

	prod := loaded.CacheForEnv("production")
	if prod.Adapter != "redis" {
		t.Errorf("prod cache Adapter = %q, want %q", prod.Adapter, "redis")
	}
	if prod.Host != "localhost" {
		t.Errorf("prod cache Host = %q, want %q", prod.Host, "localhost")
	}
}

func TestSaveAndLoadWithDatabaseBlocks(t *testing.T) {
	dir := t.TempDir()

	pf := &Polafile{
		Renderer: "react",
		Database: &Database{
			Models: "models",
			ORM:    "ent",
			Migrations: &Migrations{
				Directory: "migrations",
				Format:    "sql",
			},
			Envs: []DatabaseEnvironment{
				{Environment: "development", Adapter: "sqlite"},
				{Environment: "production", Adapter: "postgresql"},
			},
		},
	}

	if err := Save(dir, pf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Database == nil {
		t.Fatal("database block not found")
	}
	if loaded.Database.ORM != "ent" {
		t.Errorf("ORM = %q, want %q", loaded.Database.ORM, "ent")
	}
	if loaded.Database.Models != "models" {
		t.Errorf("Models = %q, want %q", loaded.Database.Models, "models")
	}

	dev := loaded.DatabaseForEnv("development")
	if dev.Adapter != "sqlite" {
		t.Errorf("dev Adapter = %q, want %q", dev.Adapter, "sqlite")
	}
	if dev.ORM != "ent" {
		t.Errorf("dev ORM = %q, want %q (inherited from base)", dev.ORM, "ent")
	}

	prod := loaded.DatabaseForEnv("production")
	if prod.Adapter != "postgresql" {
		t.Errorf("prod Adapter = %q, want %q", prod.Adapter, "postgresql")
	}

	// Helper methods use base config.
	if loaded.DatabaseModelsDir() != "models" {
		t.Errorf("DatabaseModelsDir = %q, want %q", loaded.DatabaseModelsDir(), "models")
	}
	if loaded.DatabaseORM() != "ent" {
		t.Errorf("DatabaseORM = %q, want %q", loaded.DatabaseORM(), "ent")
	}
	if loaded.DatabaseAdapter("production") != "postgresql" {
		t.Errorf("DatabaseAdapter(production) = %q, want %q", loaded.DatabaseAdapter("production"), "postgresql")
	}
}

func TestDatabaseDirectoryDefaults(t *testing.T) {
	// Empty Polafile should return new db/ defaults.
	pf := &Polafile{}
	if got := pf.DatabaseModelsDir(); got != "db/models" {
		t.Errorf("DatabaseModelsDir() = %q, want %q", got, "db/models")
	}
	if got := pf.DatabaseMigrationsDir(); got != "db/migrations" {
		t.Errorf("DatabaseMigrationsDir() = %q, want %q", got, "db/migrations")
	}
	if got := pf.DatabaseClientDir(); got != "db/client" {
		t.Errorf("DatabaseClientDir() = %q, want %q", got, "db/client")
	}
	if got := pf.DatabaseEntClientDir(); got != "db/client/ent" {
		t.Errorf("DatabaseEntClientDir() = %q, want %q", got, "db/client/ent")
	}

	// Explicit values override defaults.
	pf2 := &Polafile{
		Database: &Database{
			Models:             "custom/models",
			OrmImplementations: "custom/orm",
			Migrations:         &Migrations{Directory: "custom/migrations"},
		},
	}
	if got := pf2.DatabaseModelsDir(); got != "custom/models" {
		t.Errorf("DatabaseModelsDir() = %q, want %q", got, "custom/models")
	}
	if got := pf2.DatabaseMigrationsDir(); got != "custom/migrations" {
		t.Errorf("DatabaseMigrationsDir() = %q, want %q", got, "custom/migrations")
	}
	if got := pf2.DatabaseClientDir(); got != "custom/orm" {
		t.Errorf("DatabaseClientDir() = %q, want %q", got, "custom/orm")
	}
}

func TestCSRFEnvironmentOverride(t *testing.T) {
	pf := &Polafile{
		CSRF: &CSRF{
			Enabled: true,
			Envs: []CSRFEnvironment{
				{Environment: "test", Enabled: false},
			},
		},
	}

	if !pf.CSRFEnabled("default") {
		t.Error("CSRFEnabled(default) = false, want true")
	}
	if pf.CSRFEnabled("test") {
		t.Error("CSRFEnabled(test) = true, want false")
	}
}

func TestSecurityHeadersEnvironmentOverride(t *testing.T) {
	pf := &Polafile{
		SecurityHeaders: &SecurityHeaders{
			Enabled: true,
			Envs: []SecurityHeadersEnvironment{
				{Environment: "test", Enabled: false},
			},
		},
	}

	if !pf.SecurityHeadersEnabled("default") {
		t.Error("SecurityHeadersEnabled(default) = false, want true")
	}
	if pf.SecurityHeadersEnabled("test") {
		t.Error("SecurityHeadersEnabled(test) = true, want false")
	}
}

func TestCacheEnabledEnvironmentOverride(t *testing.T) {
	disabled := false
	pf := &Polafile{
		Cache: &Cache{
			Enabled: true,
			Adapter: "memory",
			Envs: []CacheEnvironment{
				{Environment: "test", Enabled: &disabled},
			},
		},
	}

	if !pf.CacheEnabled("default") {
		t.Error("CacheEnabled(default) = false, want true")
	}
	if pf.CacheEnabled("test") {
		t.Error("CacheEnabled(test) = true, want false")
	}
}

func TestCSRFNilDefaultsTrue(t *testing.T) {
	pf := &Polafile{}
	if !pf.CSRFEnabled("default") {
		t.Error("CSRFEnabled with nil CSRF should default to true")
	}
	if !pf.SecurityHeadersEnabled("default") {
		t.Error("SecurityHeadersEnabled with nil SecurityHeaders should default to true")
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	pf, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if pf != nil {
		t.Error("expected nil for missing file")
	}
}

func TestLoadInvalidSyntax(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, Filename), []byte(`pola {`), 0o644)

	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for invalid syntax")
	}
}

func TestLoadMissingPolaBlock(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, Filename), []byte(`other { key = "val" }`+"\n"), 0o644)

	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for missing pola block")
	}
}

func TestSaveOmitsEmptyFields(t *testing.T) {
	dir := t.TempDir()

	pf := &Polafile{
		Renderer: "react",
		Engine:   "goja",
		// Other fields intentionally empty.
	}

	if err := Save(dir, pf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, Filename))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "bundler") {
		t.Error("expected empty bundler to be omitted")
	}
	if !strings.Contains(content, "renderer") {
		t.Error("expected renderer to be present")
	}
}
