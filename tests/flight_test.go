package tests

import (
	"strings"
	"testing"
)

func walkFlight(v any, target string) bool {
	switch val := v.(type) {
	case string:
		return strings.Contains(val, target)
	case []any:
		for _, item := range val {
			if walkFlight(item, target) {
				return true
			}
		}
	case map[string]any:
		for k, item := range val {
			if strings.Contains(k, target) || walkFlight(item, target) {
				return true
			}
		}
	}
	return false
}

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
