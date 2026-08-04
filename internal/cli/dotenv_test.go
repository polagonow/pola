package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := `# a comment
export EXPORTED=yes

PLAIN=value
QUOTED="hello world"
SINGLE='literal $NOPE'
ESCAPED="line1\nline2"
INLINE=bar # trailing comment
HASHVAL=pa#ss
SPACED =  trimmed
EMPTY=
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	vars, err := parseEnvFile(path)
	if err != nil {
		t.Fatalf("parseEnvFile: %v", err)
	}

	want := map[string]string{
		"EXPORTED": "yes",
		"PLAIN":    "value",
		"QUOTED":   "hello world",
		"SINGLE":   "literal $NOPE",
		"ESCAPED":  "line1\nline2",
		"INLINE":   "bar",
		"HASHVAL":  "pa#ss",
		"SPACED":   "trimmed",
		"EMPTY":    "",
	}
	for k, wv := range want {
		if got, ok := vars[k]; !ok || got != wv {
			t.Errorf("%s = %q (ok=%v), want %q", k, got, ok, wv)
		}
	}
	if len(vars) != len(want) {
		t.Errorf("got %d vars, want %d: %v", len(vars), len(want), vars)
	}
}

func TestParseEnvFileMissing(t *testing.T) {
	_, err := parseEnvFile(filepath.Join(t.TempDir(), "nope.env"))
	if !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}

func TestLoadDotenvFilePrecedence(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".env", "FILEPREC_KEY=base\nBASE_ONLY=1\n")
	write(t, dir, ".env.development.local", "FILEPREC_KEY=specific\n")

	t.Cleanup(func() {
		os.Unsetenv("FILEPREC_KEY")
		os.Unsetenv("BASE_ONLY")
	})

	loaded, err := loadDotenv(dir)
	if err != nil {
		t.Fatalf("loadDotenv: %v", err)
	}

	// Most specific file (last in order) wins.
	if got := os.Getenv("FILEPREC_KEY"); got != "specific" {
		t.Errorf("FILEPREC_KEY = %q, want specific", got)
	}
	if got := os.Getenv("BASE_ONLY"); got != "1" {
		t.Errorf("BASE_ONLY = %q, want 1", got)
	}
	if len(loaded) != 2 {
		t.Errorf("loaded = %v, want 2 files", loaded)
	}
}

func TestLoadDotenvShellWins(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".env", "SHELLWINS_KEY=fromfile\n")

	t.Setenv("SHELLWINS_KEY", "fromshell")

	if _, err := loadDotenv(dir); err != nil {
		t.Fatalf("loadDotenv: %v", err)
	}
	if got := os.Getenv("SHELLWINS_KEY"); got != "fromshell" {
		t.Errorf("SHELLWINS_KEY = %q, want fromshell (shell must win)", got)
	}
}

func TestLoadDotenvNoFiles(t *testing.T) {
	loaded, err := loadDotenv(t.TempDir())
	if err != nil {
		t.Fatalf("loadDotenv: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("loaded = %v, want none", loaded)
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
