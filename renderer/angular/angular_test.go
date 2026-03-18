package angular_test

import (
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/renderer/angular"
)

func TestRendererName(t *testing.T) {
	r := angular.New()
	if got := r.Name(); got != "angular" {
		t.Errorf("Name() = %q, want %q", got, "angular")
	}
}

func TestRendererFileExtensions(t *testing.T) {
	r := angular.New()
	exts := r.FileExtensions()
	want := map[string]bool{".html": true, ".ts": true}
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
	var _ core.Renderer = angular.New()
}
