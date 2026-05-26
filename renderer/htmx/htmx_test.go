package htmx_test

import (
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/renderer/htmx"
)

func TestRendererName(t *testing.T) {
	r := htmx.New()
	if got := r.Name(); got != "htmx" {
		t.Errorf("Name() = %q, want %q", got, "htmx")
	}
}

func TestRendererFileExtensions(t *testing.T) {
	r := htmx.New()
	exts := r.FileExtensions()
	want := map[string]bool{".html": true, ".templ": true}
	if len(exts) != len(want) {
		t.Fatalf("FileExtensions() returned %d extensions, want %d", len(exts), len(want))
	}
	for _, ext := range exts {
		if !want[ext] {
			t.Errorf("unexpected extension %q in FileExtensions()", ext)
		}
	}
}

func TestRendererImplementsInterface(t *testing.T) {
	var _ core.Renderer = htmx.New()
}
