package layered

import (
	"errors"
	"testing"
	"time"

	"github.com/polagonow/pola/core"
)

// stubFS is a minimal in-memory core.FS for testing.
type stubFS struct {
	files map[string][]byte
	dirs  map[string][]core.FSFileInfo
}

func (s *stubFS) Name() string { return "stub" }
func (s *stubFS) ReadFile(path string) ([]byte, error) {
	data, ok := s.files[path]
	if !ok {
		return nil, errors.New("not found: " + path)
	}
	return data, nil
}
func (s *stubFS) ReadDir(path string) ([]core.FSFileInfo, error) {
	entries, ok := s.dirs[path]
	if !ok {
		return nil, errors.New("not found: " + path)
	}
	return entries, nil
}
func (s *stubFS) Exists(path string) bool {
	_, ok := s.files[path]
	return ok
}
func (s *stubFS) Watch(_ string, _ func(string)) error { return nil }

func TestReadFile_Virtual(t *testing.T) {
	inner := &stubFS{files: map[string][]byte{
		"/app/real.tsx": []byte("real content"),
	}}
	fs := New(inner)
	fs.Set("/app/virtual.tsx", []byte("virtual content"))

	data, err := fs.ReadFile("/app/virtual.tsx")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "virtual content" {
		t.Fatalf("got %q, want %q", data, "virtual content")
	}
}

func TestReadFile_FallsThrough(t *testing.T) {
	inner := &stubFS{files: map[string][]byte{
		"/app/real.tsx": []byte("real content"),
	}}
	fs := New(inner)

	data, err := fs.ReadFile("/app/real.tsx")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "real content" {
		t.Fatalf("got %q, want %q", data, "real content")
	}
}

func TestReadFile_VirtualOverridesInner(t *testing.T) {
	inner := &stubFS{files: map[string][]byte{
		"/app/page.tsx": []byte("old"),
	}}
	fs := New(inner)
	fs.Set("/app/page.tsx", []byte("new"))

	data, err := fs.ReadFile("/app/page.tsx")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("got %q, want %q", data, "new")
	}
}

func TestExists_Virtual(t *testing.T) {
	inner := &stubFS{files: map[string][]byte{}}
	fs := New(inner)
	fs.Set("/app/gen.tsx", []byte("x"))

	if !fs.Exists("/app/gen.tsx") {
		t.Fatal("expected virtual file to exist")
	}
	if fs.Exists("/app/nope.tsx") {
		t.Fatal("expected non-existent file to not exist")
	}
}

func TestRemove(t *testing.T) {
	inner := &stubFS{files: map[string][]byte{}}
	fs := New(inner)
	fs.Set("/app/temp.tsx", []byte("temp"))

	if !fs.Exists("/app/temp.tsx") {
		t.Fatal("expected file to exist after Set")
	}

	fs.Remove("/app/temp.tsx")

	if fs.Exists("/app/temp.tsx") {
		t.Fatal("expected file to not exist after Remove")
	}
}

func TestReadDir_MergesVirtual(t *testing.T) {
	inner := &stubFS{
		files: map[string][]byte{},
		dirs: map[string][]core.FSFileInfo{
			"/app": {
				{Name: "page.tsx", IsDir: false, Size: 10, ModTime: time.Now()},
			},
		},
	}
	fs := New(inner)
	fs.Set("/app/server-entry.tsx", []byte("entry"))

	entries, err := fs.ReadDir("/app")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}
	if !names["page.tsx"] || !names["server-entry.tsx"] {
		t.Fatalf("expected both page.tsx and server-entry.tsx, got %v", names)
	}
}

func TestVirtualPaths(t *testing.T) {
	inner := &stubFS{files: map[string][]byte{}}
	fs := New(inner)
	fs.Set("/b.tsx", []byte("b"))
	fs.Set("/a.tsx", []byte("a"))

	paths := fs.VirtualPaths()
	if len(paths) != 2 {
		t.Fatalf("got %d paths, want 2", len(paths))
	}
	if paths[0] != "/a.tsx" || paths[1] != "/b.tsx" {
		t.Fatalf("expected sorted paths, got %v", paths)
	}
}
