//go:build react

package react_test

import (
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/renderer/react"
)

func TestRendererName(t *testing.T) {
	r := react.New()
	if got := r.Name(); got != "react" {
		t.Errorf("Name() = %q, want %q", got, "react")
	}
}

func TestRendererFileExtensions(t *testing.T) {
	r := react.New()
	exts := r.FileExtensions()
	want := map[string]bool{".tsx": true, ".jsx": true, ".ts": true, ".js": true}
	if len(exts) != len(want) {
		t.Fatalf("FileExtensions() returned %d extensions, want %d", len(exts), len(want))
	}
	for _, ext := range exts {
		if !want[ext] {
			t.Errorf("unexpected extension %q in FileExtensions()", ext)
		}
	}
}

func TestRendererCapabilities(t *testing.T) {
	r := react.New()
	caps := r.Capabilities()
	want := map[core.Capability]bool{"streaming": true, "rsc": true}
	if len(caps) != len(want) {
		t.Fatalf("Capabilities() returned %d capabilities, want %d", len(caps), len(want))
	}
	for _, c := range caps {
		if !want[c] {
			t.Errorf("unexpected capability %q in Capabilities()", c)
		}
	}
}

func TestRendererImplementsInterface(t *testing.T) {
	// Compile-time check that *Renderer satisfies core.Renderer.
	var _ core.Renderer = react.New()
}
