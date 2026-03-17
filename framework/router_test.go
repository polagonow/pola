package framework

import (
	"reflect"
	"testing"
)

func TestMatchPattern_CatchAll(t *testing.T) {
	cases := []struct {
		pattern   string
		path      string
		wantMatch bool
		wantKey   string
		wantVal   any // string or []string or nil (key absent)
	}{
		// optional catch-all: matches zero segments (key absent)
		{"/docs/:...slug?", "/docs", true, "slug", nil},
		// optional catch-all: one segment
		{"/docs/:...slug?", "/docs/getting-started", true, "slug", []string{"getting-started"}},
		// optional catch-all: two segments
		{"/docs/:...slug?", "/docs/getting-started/installation", true, "slug", []string{"getting-started", "installation"}},
		// optional catch-all: three segments
		{"/docs/:...slug?", "/docs/a/b/c", true, "slug", []string{"a", "b", "c"}},

		// required catch-all: zero segments → no match
		{"/shop/:...path", "/shop", false, "", nil},
		// required catch-all: one segment
		{"/shop/:...path", "/shop/clothes", true, "path", []string{"clothes"}},
		// required catch-all: two segments
		{"/shop/:...path", "/shop/a/b", true, "path", []string{"a", "b"}},

		// mix of static prefix + catch-all
		{"/posts/:slug/tags/:...rest", "/posts/my-post/tags/go/react", true, "rest", []string{"go", "react"}},

		// regular single-param still works
		{"/posts/:slug", "/posts/hello", true, "slug", "hello"},
		{"/posts/:slug", "/posts/hello/extra", false, "", nil},
	}

	for _, tc := range cases {
		params, ok := MatchPattern(tc.pattern, tc.path)
		if ok != tc.wantMatch {
			t.Errorf("MatchPattern(%q, %q) match=%v, want %v", tc.pattern, tc.path, ok, tc.wantMatch)
			continue
		}
		if !tc.wantMatch {
			continue
		}
		if tc.wantVal == nil {
			if _, exists := params[tc.wantKey]; exists {
				t.Errorf("MatchPattern(%q, %q): expected key %q absent, got %v",
					tc.pattern, tc.path, tc.wantKey, params[tc.wantKey])
			}
			continue
		}
		got := params[tc.wantKey]
		if !reflect.DeepEqual(got, tc.wantVal) {
			t.Errorf("MatchPattern(%q, %q) params[%q] = %#v, want %#v", tc.pattern, tc.path, tc.wantKey, got, tc.wantVal)
		}
	}
}
