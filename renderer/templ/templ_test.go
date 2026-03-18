package templ_test

import (
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/renderer/templ"
)

func TestRendererName(t *testing.T) {
	r := templ.New()
	if got := r.Name(); got != "templ" {
		t.Errorf("Name() = %q, want %q", got, "templ")
	}
}

func TestRendererFileExtensions(t *testing.T) {
	r := templ.New()
	exts := r.FileExtensions()
	if len(exts) != 1 || exts[0] != ".templ" {
		t.Errorf("FileExtensions() = %v, want [.templ]", exts)
	}
}

func TestRendererImplementsInterface(t *testing.T) {
	var _ core.Renderer = templ.New()
}
