package nextjs

import (
	"reflect"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/patternmatch"
)

// ── compilePattern ──────────────────────────────────────────────────────────

func TestCompilePattern(t *testing.T) {
	cases := []struct {
		pattern    string
		isStatic   bool
		paramNames []string
		segments   []patternmatch.SegmentKind
	}{
		{"/", true, nil, []patternmatch.SegmentKind{patternmatch.SegStatic}},
		{"/about", true, nil, []patternmatch.SegmentKind{patternmatch.SegStatic}},
		{"/a/b/c", true, nil, []patternmatch.SegmentKind{patternmatch.SegStatic, patternmatch.SegStatic, patternmatch.SegStatic}},
		{"/posts/:slug", false, []string{"slug"}, []patternmatch.SegmentKind{patternmatch.SegStatic, patternmatch.SegDynamic}},
		{"/users/:id/posts/:postId", false, []string{"id", "postId"}, []patternmatch.SegmentKind{patternmatch.SegStatic, patternmatch.SegDynamic, patternmatch.SegStatic, patternmatch.SegDynamic}},
		{"/shop/:...path", false, []string{"path"}, []patternmatch.SegmentKind{patternmatch.SegStatic, patternmatch.SegCatchAll}},
		{"/docs/:...slug?", false, []string{"slug"}, []patternmatch.SegmentKind{patternmatch.SegStatic, patternmatch.SegOptCatchAll}},
		{"/posts/:slug/tags/:...rest", false, []string{"slug", "rest"}, []patternmatch.SegmentKind{patternmatch.SegStatic, patternmatch.SegDynamic, patternmatch.SegStatic, patternmatch.SegCatchAll}},
	}
	for _, tc := range cases {
		cr := compilePattern(core.Route{Pattern: tc.pattern})
		if cr.compiled.IsStatic != tc.isStatic {
			t.Errorf("compilePattern(%q).IsStatic = %v, want %v", tc.pattern, cr.compiled.IsStatic, tc.isStatic)
		}
		if !reflect.DeepEqual(cr.compiled.ParamNames, tc.paramNames) {
			t.Errorf("compilePattern(%q).ParamNames = %v, want %v", tc.pattern, cr.compiled.ParamNames, tc.paramNames)
		}
		if !reflect.DeepEqual(cr.compiled.Segments, tc.segments) {
			t.Errorf("compilePattern(%q).Segments = %v, want %v", tc.pattern, cr.compiled.Segments, tc.segments)
		}
	}
}

// ── matchCompiled ───────────────────────────────────────────────────────────

func TestMatchCompiled_Basic(t *testing.T) {
	cases := []struct {
		pattern   string
		path      string
		wantMatch bool
		wantKey   string
		wantVal   any
	}{
		{"/about", "/about", true, "", nil},
		{"/about", "/contact", false, "", nil},
		{"/posts/:slug", "/posts/hello", true, "slug", "hello"},
		{"/posts/:slug", "/posts/hello/extra", false, "", nil},
		{"/shop/:...path", "/shop/a/b", true, "path", []string{"a", "b"}},
		{"/shop/:...path", "/shop", false, "", nil},
		{"/docs/:...slug?", "/docs", true, "slug", nil},
		{"/docs/:...slug?", "/docs/a/b", true, "slug", []string{"a", "b"}},
		{"/", "/", true, "", nil},
	}
	for _, tc := range cases {
		cr := compilePattern(core.Route{Pattern: tc.pattern})
		params, ok := matchCompiled(cr, tc.path)
		if ok != tc.wantMatch {
			t.Errorf("matchCompiled(%q, %q) = %v, want %v", tc.pattern, tc.path, ok, tc.wantMatch)
			continue
		}
		if !ok || tc.wantKey == "" {
			continue
		}
		if tc.wantVal == nil {
			if _, exists := params[tc.wantKey]; exists {
				t.Errorf("matchCompiled(%q, %q): expected key %q absent", tc.pattern, tc.path, tc.wantKey)
			}
		} else if !reflect.DeepEqual(params[tc.wantKey], tc.wantVal) {
			t.Errorf("matchCompiled(%q, %q) params[%q] = %#v, want %#v", tc.pattern, tc.path, tc.wantKey, params[tc.wantKey], tc.wantVal)
		}
	}
}

func TestMatchCompiled_URIDecoding(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		key     string
		want    any
	}{
		{"/posts/:slug", "/posts/hello%20world", "slug", "hello world"},
		{"/posts/:slug", "/posts/caf%C3%A9", "slug", "café"},
		{"/shop/:...path", "/shop/a%20b/c%20d", "path", []string{"a b", "c d"}},
	}
	for _, tc := range cases {
		cr := compilePattern(core.Route{Pattern: tc.pattern})
		params, ok := matchCompiled(cr, tc.path)
		if !ok {
			t.Errorf("matchCompiled(%q, %q): expected match", tc.pattern, tc.path)
			continue
		}
		if !reflect.DeepEqual(params[tc.key], tc.want) {
			t.Errorf("matchCompiled(%q, %q) params[%q] = %#v, want %#v", tc.pattern, tc.path, tc.key, params[tc.key], tc.want)
		}
	}
}

// ── sortCompiledRoutes ──────────────────────────────────────────────────────

func TestSortCompiledRoutes_SegmentBySegment(t *testing.T) {
	routes := []*compiledRoute{
		compilePattern(core.Route{Pattern: "/docs/:...slug?", Export: "DocsOptional"}),
		compilePattern(core.Route{Pattern: "/docs/:...slug", Export: "DocsCatchAll"}),
		compilePattern(core.Route{Pattern: "/docs/:id", Export: "DocsId"}),
	}
	sortCompiledRoutes(routes)

	want := []string{"DocsId", "DocsCatchAll", "DocsOptional"}
	for i, w := range want {
		if routes[i].Export != w {
			t.Errorf("position %d: got %s, want %s", i, routes[i].Export, w)
		}
	}
}

func TestSortCompiledRoutes_SameScoreDifferentSegments(t *testing.T) {
	routes := []*compiledRoute{
		compilePattern(core.Route{Pattern: "/:slug/details", Export: "SlugDetails"}),
		compilePattern(core.Route{Pattern: "/api/:id", Export: "ApiId"}),
	}
	sortCompiledRoutes(routes)

	if routes[0].Export != "ApiId" {
		t.Errorf("expected ApiId first, got %s", routes[0].Export)
	}
	if routes[1].Export != "SlugDetails" {
		t.Errorf("expected SlugDetails second, got %s", routes[1].Export)
	}
}

// ── normalizePath ───────────────────────────────────────────────────────────

func TestNormalizePath(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"/", "/"},
		{"/about", "/about"},
		{"/about/", "/about"},
		{"/a/b/c/", "/a/b/c"},
		{"", ""},
	}
	for _, tc := range cases {
		got := normalizePath(tc.input)
		if got != tc.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

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
		wantVal   any
	}{
		{"/docs/:...slug?", "/docs", true, "slug", nil},
		{"/docs/:...slug?", "/docs/getting-started", true, "slug", []string{"getting-started"}},
		{"/docs/:...slug?", "/docs/getting-started/installation", true, "slug", []string{"getting-started", "installation"}},
		{"/docs/:...slug?", "/docs/a/b/c", true, "slug", []string{"a", "b", "c"}},
		{"/shop/:...path", "/shop", false, "", nil},
		{"/shop/:...path", "/shop/clothes", true, "path", []string{"clothes"}},
		{"/shop/:...path", "/shop/a/b", true, "path", []string{"a", "b"}},
		{"/posts/:slug/tags/:...rest", "/posts/my-post/tags/go/react", true, "rest", []string{"go", "react"}},
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
	cases := []struct {
		pattern string
		want    int
	}{
		{"/", 2},
		{"/about", 2},
		{"/a/b/c", 4},
		{"/posts/:slug", 2},
		{"/posts/:...path", -8},
		{"/docs/:...slug?", -18},
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
