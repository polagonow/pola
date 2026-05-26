package osfs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/polagonow/pola/fs/osfs"
)

func TestName(t *testing.T) {
	fs := osfs.New(".")
	if fs.Name() != "osfs" {
		t.Fatalf("expected osfs, got %s", fs.Name())
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	fs := osfs.New(dir)

	// A file that does not exist.
	if fs.Exists("nonexistent.txt") {
		t.Fatal("Exists should return false for missing file")
	}

	// Create a file.
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fs.Exists("hello.txt") {
		t.Fatal("Exists should return true for existing file")
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	fs := osfs.New(dir)

	content := []byte("hello pola")
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := fs.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("expected %q, got %q", content, got)
	}
}

func TestReadDir(t *testing.T) {
	dir := t.TempDir()
	fs := osfs.New(dir)

	// Write two files and one sub-directory.
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := fs.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}
	for _, expected := range []string{"a.txt", "b.txt", "sub"} {
		if !names[expected] {
			t.Fatalf("missing entry %q", expected)
		}
	}
}

func TestReadFile_Missing(t *testing.T) {
	fs := osfs.New(t.TempDir())
	_, err := fs.ReadFile("does-not-exist.txt")
	if err == nil {
		t.Fatal("expected error reading missing file")
	}
}
