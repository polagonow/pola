package hybrid_test

import (
	"errors"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/fs/hybrid"
)

// stubFS is a simple in-memory FS for testing.
type stubFS struct {
	files map[string][]byte
	name  string
}

func newStub(name string, files map[string][]byte) *stubFS {
	return &stubFS{name: name, files: files}
}

func (s *stubFS) Name() string { return s.name }

func (s *stubFS) ReadFile(path string) ([]byte, error) {
	if data, ok := s.files[path]; ok {
		return data, nil
	}
	return nil, errors.New("not found: " + path)
}

func (s *stubFS) ReadDir(path string) ([]core.FSFileInfo, error) {
	if _, ok := s.files[path]; ok {
		return []core.FSFileInfo{{Name: path}}, nil
	}
	return nil, errors.New("not found: " + path)
}

func (s *stubFS) Exists(path string) bool {
	_, ok := s.files[path]
	return ok
}

func (s *stubFS) Watch(path string, onChange func(string)) error { return nil }

func TestHybrid_PrefersEmbed(t *testing.T) {
	embedFS := newStub("embed", map[string][]byte{
		"file.txt": []byte("from embed"),
	})
	osFS := newStub("os", map[string][]byte{
		"file.txt": []byte("from os"),
	})

	h := hybrid.New(embedFS, osFS)

	data, err := h.ReadFile("file.txt")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(data) != "from embed" {
		t.Fatalf("expected 'from embed', got %q", string(data))
	}
}

func TestHybrid_FallsBackToOS(t *testing.T) {
	embedFS := newStub("embed", map[string][]byte{})
	osFS := newStub("os", map[string][]byte{
		"file.txt": []byte("from os"),
	})

	h := hybrid.New(embedFS, osFS)

	data, err := h.ReadFile("file.txt")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(data) != "from os" {
		t.Fatalf("expected 'from os', got %q", string(data))
	}
}

func TestHybrid_Exists(t *testing.T) {
	embedFS := newStub("embed", map[string][]byte{"e.txt": {}})
	osFS := newStub("os", map[string][]byte{"o.txt": {}})

	h := hybrid.New(embedFS, osFS)

	if !h.Exists("e.txt") {
		t.Fatal("should exist in embed")
	}
	if !h.Exists("o.txt") {
		t.Fatal("should exist in os fallback")
	}
	if h.Exists("missing.txt") {
		t.Fatal("should not exist")
	}
}

func TestHybrid_Name(t *testing.T) {
	h := hybrid.New(newStub("e", nil), newStub("o", nil))
	if h.Name() != "hybrid" {
		t.Fatalf("expected hybrid, got %s", h.Name())
	}
}

func TestHybrid_ReadDir_PrefersEmbed(t *testing.T) {
	embedFS := newStub("embed", map[string][]byte{"dir": {}})
	osFS := newStub("os", map[string][]byte{})

	h := hybrid.New(embedFS, osFS)
	entries, err := h.ReadDir("dir")
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries from embed")
	}
}
