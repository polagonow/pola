package nextjs

import (
	"reflect"
	"testing"

	"github.com/polagonow/pola/core"
)

// ── MatchPattern ─────────────────────────────────────────────────────────────

func TestMatchPattern_Static(t *testing.T) {
	cases := []struct {
		pattern   string
		path      string
		wantMatch bool
	}{
		{"/", "/", true},
		{"/about", "/about", true},
		{"/about", "/contact", false},
		{"/a/b/c", "/a/b/c", true},
		{"/a/b/c", "/a/b", false},
		{"/a/b/c", "/a/b/c/d", false},
	}
	for _, tc := range cases {
		params, ok := MatchPattern(tc.pattern, tc.path)
		if ok != tc.wantMatch {
			t.Errorf("MatchPattern(%q, %q) match=%v want %v", tc.pattern, tc.path, ok, tc.wantMatch)
		}
		if ok && params == nil {
			t.Errorf("MatchPattern(%q, %q) returned nil params on match", tc.pattern, tc.path)
		}
	}
}

func TestMatchPattern_Dynamic(t *testing.T) {
	cases := []struct {
		pattern   string
		path      string
		wantMatch bool
		wantKey   string
		wantVal   string
	}{
		{"/posts/:slug", "/posts/hello", true, "slug", "hello"},
		{"/posts/:slug", "/posts/", true, "slug", ""},
		{"/posts/:slug", "/posts/hello/extra", false, "", ""},
		{"/products/:id/reviews", "/products/42/reviews", true, "id", "42"},
		{"/products/:id/reviews", "/products/42/other", false, "", ""},
	}
	for _, tc := range cases {
		params, ok := MatchPattern(tc.pattern, tc.path)
		if ok != tc.wantMatch {
			t.Errorf("MatchPattern(%q, %q) match=%v want %v", tc.pattern, tc.path, ok, tc.wantMatch)
			continue
		}
		if !ok {
			continue
		}
		if got, exists := params[tc.wantKey]; !exists || got != tc.wantVal {
			t.Errorf("MatchPattern(%q, %q) params[%q]=%v want %q", tc.pattern, tc.path, tc.wantKey, got, tc.wantVal)
		}
	}
}

func TestMatchPattern_CatchAll(t *testing.T) {
	cases := []struct {
		pattern   string
		path      string
		wantMatch bool
		wantKey   string
		wantVal   any // string | []string | nil (key absent)
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
			t.Errorf("MatchPattern(%q, %q) params[%q] = %#v, want %#v",
				tc.pattern, tc.path, tc.wantKey, got, tc.wantVal)
		}
	}
}

func TestMatchPattern_MultipleParams(t *testing.T) {
	params, ok := MatchPattern("/users/:userId/posts/:postId", "/users/42/posts/99")
	if !ok {
		t.Fatal("expected match")
	}
	if params["userId"] != "42" {
		t.Errorf("userId=%v want 42", params["userId"])
	}
	if params["postId"] != "99" {
		t.Errorf("postId=%v want 99", params["postId"])
	}
}

func TestMatchPattern_RootPattern(t *testing.T) {
	params, ok := MatchPattern("/", "/")
	if !ok {
		t.Fatal("expected match for root pattern")
	}
	if len(params) != 0 {
		t.Errorf("expected empty params for root, got %v", params)
	}
}

// ── routeScore ────────────────────────────────────────────────────────────────

func TestRouteScore(t *testing.T) {
	// routeScore counts static segments as +1, :... as -10, :...? as -20.
	// strings.Split("/x/y", "/") = ["","x","y"] so every pattern has at least
	// one empty static segment from the leading slash.
	cases := []struct {
		pattern string
		want    int
	}{
		{"/", 2},                 // ["",""] → 2 static
		{"/about", 2},            // ["","about"] → 2 static
		{"/a/b/c", 4},            // ["","a","b","c"] → 4 static
		{"/posts/:slug", 2},      // ["","posts",":slug"] → 2 static + 1 neutral = 2
		{"/posts/:...path", -8},  // ["","posts",":...path"] → 2 static - 10 = -8
		{"/docs/:...slug?", -18}, // ["","docs",":...slug?"] → 2 static - 20 = -18
	}
	for _, tc := range cases {
		got := routeScore(tc.pattern)
		if got != tc.want {
			t.Errorf("routeScore(%q) = %d, want %d", tc.pattern, got, tc.want)
		}
	}
}

// ── sortRoutes ────────────────────────────────────────────────────────────────

func TestSortRoutes_Priority(t *testing.T) {
	routes := []core.Route{
		{Pattern: "/docs/:...slug?", Export: "DocsOptional"},
		{Pattern: "/docs/:...slug", Export: "DocsCatchAll"},
		{Pattern: "/docs/:id", Export: "DocsId"},
		{Pattern: "/docs/intro", Export: "DocsIntro"},
	}
	sortRoutes(routes)

	// Static first, then dynamic, then catch-all, then optional catch-all.
	if routes[0].Export != "DocsIntro" {
		t.Errorf("first route should be DocsIntro (static), got %s", routes[0].Export)
	}
	if routes[1].Export != "DocsId" {
		t.Errorf("second route should be DocsId (dynamic), got %s", routes[1].Export)
	}
	if routes[2].Export != "DocsCatchAll" {
		t.Errorf("third route should be DocsCatchAll, got %s", routes[2].Export)
	}
	if routes[3].Export != "DocsOptional" {
		t.Errorf("fourth route should be DocsOptional, got %s", routes[3].Export)
	}
}
