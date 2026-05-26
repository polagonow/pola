package vue_test

import (
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/renderer/vue"
)

func TestRendererName(t *testing.T) {
	r := vue.New()
	if got := r.Name(); got != "vue" {
		t.Errorf("Name() = %q, want %q", got, "vue")
	}
}

func TestRendererFileExtensions(t *testing.T) {
	r := vue.New()
	exts := r.FileExtensions()
	if len(exts) != 1 || exts[0] != ".vue" {
		t.Errorf("FileExtensions() = %v, want [.vue]", exts)
	}
}

func TestRendererImplementsInterface(t *testing.T) {
	var _ core.Renderer = vue.New()
}
