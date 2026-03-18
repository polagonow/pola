//go:build tailwind

package tailwind

import (
	"context"
	"os/exec"
	"testing"
)

func TestName(t *testing.T) {
	tw := New()
	if tw.Name() != "tailwind" {
		t.Fatalf("expected 'tailwind', got %q", tw.Name())
	}
}

func TestNew_DefaultBin(t *testing.T) {
	tw := New()
	if tw.Bin != "tailwindcss" {
		t.Fatalf("expected default Bin 'tailwindcss', got %q", tw.Bin)
	}
}

func TestProcess_SkipsIfNotInstalled(t *testing.T) {
	if _, err := exec.LookPath("tailwindcss"); err != nil {
		t.Skip("tailwindcss not installed — skipping exec test")
	}

	tw := New()
	// This will fail because the input file doesn't exist, but should not panic.
	err := tw.Process(context.Background(), "/nonexistent/input.css", "/tmp/output.css")
	if err == nil {
		t.Fatal("expected an error for a non-existent input file")
	}
}

func TestWatch_SkipsIfNotInstalled(t *testing.T) {
	if _, err := exec.LookPath("tailwindcss"); err != nil {
		t.Skip("tailwindcss not installed — skipping exec test")
	}

	tw := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so the watcher exits
	// Should not panic or block.
	_ = tw.Watch(ctx, "/nonexistent/input.css", "/tmp/output.css", func() {})
}
