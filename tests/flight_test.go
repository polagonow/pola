package tests

import (
	"testing"
)

func TestFlightTree_Root(t *testing.T) {
	body := rsc(t, "/")
	tree := flightTree(t, body)
	arr, ok := tree.([]any)
	if !ok || len(arr) < 2 {
		t.Fatalf("expected array root, got %T: %v", tree, tree)
	}
	if arr[0] != "$" {
		t.Errorf("expected '$' as first element, got %v", arr[0])
	}
}

func TestFlightTree_HomePage_HasBranding(t *testing.T) {
	body := rsc(t, "/")
	// Async components land in later Flight rows, not row 0 — use raw body check.
	for _, want := range []string{"Dev", "Blog", "Read posts"} {
		if !flightContains(body, want) {
			t.Errorf("HomePage Flight missing branding text %q", want)
		}
	}
}
