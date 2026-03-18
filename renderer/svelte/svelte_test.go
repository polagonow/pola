package svelte_test

import (
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/renderer/svelte"
)

func TestRendererName(t *testing.T) {
	r := svelte.New()
	if got := r.Name(); got != "svelte" {
		t.Errorf("Name() = %q, want %q", got, "svelte")
	}
}

func TestRendererFileExtensions(t *testing.T) {
	r := svelte.New()
	exts := r.FileExtensions()
	if len(exts) != 1 || exts[0] != ".svelte" {
		t.Errorf("FileExtensions() = %v, want [.svelte]", exts)
	}
}

func TestRendererImplementsInterface(t *testing.T) {
	var _ core.Renderer = svelte.New()
}
