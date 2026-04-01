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
		Version:        "0.1.0",
		Renderer:       "react@^19.0.0",
		Engine:         "goja@0.0.0-20240220",
		Bundler:        "esbuild@^0.21.0",
		Router:         "nextjs",
		CSS:            "tailwind@^4.0.0",
		Cache:          "memory",
		PackageManager: "pnpm@^9.0.0",
		AppDir:         "app",
		ActionsDir:     "actions",
		RoutesDir:      "routes",
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
	if loaded.Cache != "memory" {
		t.Errorf("Cache = %q, want %q", loaded.Cache, "memory")
	}
	if loaded.PackageManager != "pnpm@^9.0.0" {
		t.Errorf("PackageManager = %q, want %q", loaded.PackageManager, "pnpm@^9.0.0")
	}
	if loaded.AppDir != "app" {
		t.Errorf("AppDir = %q, want %q", loaded.AppDir, "app")
	}
	if loaded.ActionsDir != "actions" {
		t.Errorf("ActionsDir = %q, want %q", loaded.ActionsDir, "actions")
	}
	if loaded.RoutesDir != "routes" {
		t.Errorf("RoutesDir = %q, want %q", loaded.RoutesDir, "routes")
	}

	// Verify ParseVersioned works with loaded values.
	name, ver := ParseVersioned(loaded.CSS)
	if name != "tailwind" || ver != "^4.0.0" {
		t.Errorf("ParseVersioned(CSS) = (%q, %q), want (\"tailwind\", \"^4.0.0\")", name, ver)
	}
}

func TestSaveAndLoadWithEnvBlocks(t *testing.T) {
	dir := t.TempDir()

	pf := &Polafile{
		Renderer:       "react",
		Engine:         "goja",
		Bundler:        "esbuild",
		PackageManager: "pnpm",
		Development: &Environment{
			Bundler: "esbuild",
		},
		Production: &Environment{
			Bundler: "webpack",
			CSS:     "postcss",
		},
	}

	if err := Save(dir, pf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("Load returned nil")
	}

	if loaded.Production == nil {
		t.Fatal("Production block is nil")
	}
	if loaded.Production.Bundler != "webpack" {
		t.Errorf("Production.Bundler = %q, want %q", loaded.Production.Bundler, "webpack")
	}
	if loaded.Production.CSS != "postcss" {
		t.Errorf("Production.CSS = %q, want %q", loaded.Production.CSS, "postcss")
	}
}

func TestForEnv(t *testing.T) {
	pf := &Polafile{
		Renderer:       "react",
		Engine:         "goja",
		Bundler:        "esbuild",
		PackageManager: "pnpm",
		Production: &Environment{
			Bundler: "webpack",
			CSS:     "postcss",
		},
	}

	dev := pf.ForEnv("development")
	if dev.Bundler != "esbuild" {
		t.Errorf("dev Bundler = %q, want %q", dev.Bundler, "esbuild")
	}

	prod := pf.ForEnv("production")
	if prod.Bundler != "webpack" {
		t.Errorf("prod Bundler = %q, want %q", prod.Bundler, "webpack")
	}
	if prod.CSS != "postcss" {
		t.Errorf("prod CSS = %q, want %q", prod.CSS, "postcss")
	}
	if prod.Renderer != "react" {
		t.Errorf("prod Renderer = %q, want %q (inherited from base)", prod.Renderer, "react")
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
