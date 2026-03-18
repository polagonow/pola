package globals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolyfillGlobalsAreStable(t *testing.T) {
	t.Parallel()

	// These JS polyfills are the canonical source for certain globals; this test
	// ensures we don't accidentally rename the Go constants (or vice versa).
	cases := []struct {
		relPath string
		want    []string
	}{
		{
			relPath: filepath.FromSlash("vm/polyfill/01_microtask.js"),
			want:    []string{DrainMicrotasksFn},
		},
		{
			relPath: filepath.FromSlash("vm/polyfill/04_readablestream.js"),
			want:    []string{PullStreamFn, DrainMicrotasksFn},
		},
		{
			relPath: filepath.FromSlash("vm/polyfill/05_webpackrequire.js"),
			want:    []string{WebpackRequireFn, WebpackChunkLoadFn, WebpackModuleRegistry},
		},
	}

	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// tests in this package run from framework/globals; repo root is two levels up.
	repoRoot := filepath.Clean(filepath.Join(root, "..", ".."))

	for _, tc := range cases {
		tc := tc
		t.Run(tc.relPath, func(t *testing.T) {
			t.Parallel()

			abs := filepath.Join(repoRoot, tc.relPath)
			b, err := os.ReadFile(abs)
			if err != nil {
				t.Fatalf("read %s: %v", tc.relPath, err)
			}
			src := string(b)
			for _, w := range tc.want {
				if !strings.Contains(src, w) {
					t.Fatalf("%s: expected to contain %q", tc.relPath, w)
				}
			}
		})
	}
}
