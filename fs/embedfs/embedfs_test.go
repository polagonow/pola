package embedfs_test

import (
	"embed"
	"testing"

	"github.com/polagonow/pola/fs/embedfs"
)

//go:embed testdata/hello.txt
var testFS embed.FS

func TestName(t *testing.T) {
	e := embedfs.New(testFS)
	if e.Name() != "embedfs" {
		t.Fatalf("expected embedfs, got %s", e.Name())
	}
}

func TestReadFile(t *testing.T) {
	e := embedfs.New(testFS)
	data, err := e.ReadFile("testdata/hello.txt")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(data) != "hello embed\n" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

func TestReadDir(t *testing.T) {
	e := embedfs.New(testFS)
	entries, err := e.ReadDir("testdata")
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "hello.txt" {
		t.Fatalf("unexpected entry name: %s", entries[0].Name)
	}
}

func TestExists(t *testing.T) {
	e := embedfs.New(testFS)
	if !e.Exists("testdata/hello.txt") {
		t.Fatal("Exists should be true for embedded file")
	}
	if e.Exists("testdata/nope.txt") {
		t.Fatal("Exists should be false for missing file")
	}
}

func TestWatch_NoOp(t *testing.T) {
	e := embedfs.New(testFS)
	if err := e.Watch("testdata", func(string) {}); err != nil {
		t.Fatalf("Watch should be a no-op, got: %v", err)
	}
}
