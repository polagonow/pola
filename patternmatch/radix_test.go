package patternmatch

import (
	"fmt"
	"reflect"
	"testing"
)

// ── Static routes ───────────────────────────────────────────────────────────

func TestRouteTable_StaticRoot(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/"),
	})
	r, params, ok := rt.Match("/")
	if !ok {
		t.Fatal("expected match for /")
	}
	if r.Pattern != "/" {
		t.Errorf("pattern = %q, want /", r.Pattern)
	}
	if len(params) != 0 {
		t.Errorf("expected empty params, got %v", params)
	}
}

func TestRouteTable_StaticPaths(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/posts"),
		CompilePattern("/posts/new"),
		CompilePattern("/about"),
	})

	cases := []struct {
		path    string
		want    string
		wantOK  bool
	}{
		{"/posts", "/posts", true},
		{"/posts/new", "/posts/new", true},
		{"/about", "/about", true},
		{"/missing", "", false},
	}

	for _, tc := range cases {
		r, _, ok := rt.Match(tc.path)
		if ok != tc.wantOK {
			t.Errorf("Match(%q) ok=%v, want %v", tc.path, ok, tc.wantOK)
			continue
		}
		if ok && r.Pattern != tc.want {
			t.Errorf("Match(%q) pattern=%q, want %q", tc.path, r.Pattern, tc.want)
		}
	}
}

// ── Dynamic routes ──────────────────────────────────────────────────────────

func TestRouteTable_DynamicSingle(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/posts/:slug"),
	})

	r, params, ok := rt.Match("/posts/hello")
	if !ok {
		t.Fatal("expected match")
	}
	if r.Pattern != "/posts/:slug" {
		t.Errorf("pattern = %q", r.Pattern)
	}
	if params["slug"] != "hello" {
		t.Errorf("slug = %v, want hello", params["slug"])
	}
}

func TestRouteTable_DynamicMultiple(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/users/:id/posts/:postId"),
	})

	r, params, ok := rt.Match("/users/42/posts/99")
	if !ok {
		t.Fatal("expected match")
	}
	if r.Pattern != "/users/:id/posts/:postId" {
		t.Errorf("pattern = %q", r.Pattern)
	}
	if params["id"] != "42" {
		t.Errorf("id = %v, want 42", params["id"])
	}
	if params["postId"] != "99" {
		t.Errorf("postId = %v, want 99", params["postId"])
	}
}

func TestRouteTable_DynamicNoMatch(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/posts/:slug"),
	})

	// Extra segment should not match a single dynamic param.
	_, _, ok := rt.Match("/posts/hello/extra")
	if ok {
		t.Error("expected no match for /posts/hello/extra")
	}
}

// ── Catch-all routes ────────────────────────────────────────────────────────

func TestRouteTable_CatchAll(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/files/:...path"),
	})

	cases := []struct {
		path   string
		wantOK bool
		want   []string
	}{
		{"/files/a/b/c", true, []string{"a", "b", "c"}},
		{"/files/readme.txt", true, []string{"readme.txt"}},
		{"/files", false, nil}, // catch-all requires at least one segment
	}

	for _, tc := range cases {
		r, params, ok := rt.Match(tc.path)
		if ok != tc.wantOK {
			t.Errorf("Match(%q) ok=%v, want %v", tc.path, ok, tc.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if r.Pattern != "/files/:...path" {
			t.Errorf("Match(%q) pattern=%q", tc.path, r.Pattern)
		}
		if !reflect.DeepEqual(params["path"], tc.want) {
			t.Errorf("Match(%q) path=%v, want %v", tc.path, params["path"], tc.want)
		}
	}
}

// ── Optional catch-all routes ───────────────────────────────────────────────

func TestRouteTable_OptionalCatchAll(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/docs/:...slug?"),
	})

	cases := []struct {
		path    string
		wantOK  bool
		wantKey string
		want    any
	}{
		{"/docs", true, "slug", nil},           // optional — no segments
		{"/docs/a", true, "slug", []string{"a"}},
		{"/docs/a/b/c", true, "slug", []string{"a", "b", "c"}},
	}

	for _, tc := range cases {
		r, params, ok := rt.Match(tc.path)
		if ok != tc.wantOK {
			t.Errorf("Match(%q) ok=%v, want %v", tc.path, ok, tc.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if r.Pattern != "/docs/:...slug?" {
			t.Errorf("Match(%q) pattern=%q", tc.path, r.Pattern)
		}
		if tc.want == nil {
			if _, exists := params[tc.wantKey]; exists {
				t.Errorf("Match(%q) expected no key %q, got %v", tc.path, tc.wantKey, params[tc.wantKey])
			}
		} else if !reflect.DeepEqual(params[tc.wantKey], tc.want) {
			t.Errorf("Match(%q) %s=%v, want %v", tc.path, tc.wantKey, params[tc.wantKey], tc.want)
		}
	}
}

// ── Specificity: static wins over dynamic ───────────────────────────────────

func TestRouteTable_StaticWinsOverDynamic(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/posts/:slug"),
		CompilePattern("/posts/new"),
	})

	r, _, ok := rt.Match("/posts/new")
	if !ok {
		t.Fatal("expected match")
	}
	if r.Pattern != "/posts/new" {
		t.Errorf("expected static /posts/new to win, got %q", r.Pattern)
	}

	// Dynamic should still work for other values.
	r, params, ok := rt.Match("/posts/hello")
	if !ok {
		t.Fatal("expected match for /posts/hello")
	}
	if r.Pattern != "/posts/:slug" {
		t.Errorf("pattern = %q, want /posts/:slug", r.Pattern)
	}
	if params["slug"] != "hello" {
		t.Errorf("slug = %v", params["slug"])
	}
}

// ── Specificity: dynamic wins over catch-all ────────────────────────────────

func TestRouteTable_DynamicWinsOverCatchAll(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/docs/:...slug"),
		CompilePattern("/docs/:id"),
	})

	// Single segment should match dynamic, not catch-all.
	r, params, ok := rt.Match("/docs/intro")
	if !ok {
		t.Fatal("expected match")
	}
	if r.Pattern != "/docs/:id" {
		t.Errorf("expected dynamic /docs/:id to win, got %q", r.Pattern)
	}
	if params["id"] != "intro" {
		t.Errorf("id = %v", params["id"])
	}

	// Multiple segments should match catch-all.
	r, params, ok = rt.Match("/docs/a/b")
	if !ok {
		t.Fatal("expected match for /docs/a/b")
	}
	if r.Pattern != "/docs/:...slug" {
		t.Errorf("expected catch-all, got %q", r.Pattern)
	}
	if !reflect.DeepEqual(params["slug"], []string{"a", "b"}) {
		t.Errorf("slug = %v", params["slug"])
	}
}

// ── Specificity: catch-all wins over optional catch-all ─────────────────────

func TestRouteTable_CatchAllWinsOverOptCatchAll(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/files/:...path?"),
		CompilePattern("/files/:...path"),
	})

	// With segments, mandatory catch-all should win.
	r, _, ok := rt.Match("/files/a/b")
	if !ok {
		t.Fatal("expected match")
	}
	if r.Pattern != "/files/:...path" {
		t.Errorf("expected mandatory catch-all to win, got %q", r.Pattern)
	}

	// Without segments, optional catch-all should match.
	r, _, ok = rt.Match("/files")
	if !ok {
		t.Fatal("expected match for /files")
	}
	if r.Pattern != "/files/:...path?" {
		t.Errorf("expected optional catch-all for /files, got %q", r.Pattern)
	}
}

// ── Overlapping routes ──────────────────────────────────────────────────────

func TestRouteTable_OverlappingRoutes(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/posts/:slug"),
		CompilePattern("/posts/new"),
		CompilePattern("/posts/:slug/comments"),
		CompilePattern("/posts/:slug/comments/:commentId"),
	})

	cases := []struct {
		path       string
		wantPat    string
		wantParams map[string]any
	}{
		{"/posts/new", "/posts/new", map[string]any{}},
		{"/posts/hello", "/posts/:slug", map[string]any{"slug": "hello"}},
		{"/posts/hello/comments", "/posts/:slug/comments", map[string]any{"slug": "hello"}},
		{"/posts/hello/comments/5", "/posts/:slug/comments/:commentId", map[string]any{"slug": "hello", "commentId": "5"}},
	}

	for _, tc := range cases {
		r, params, ok := rt.Match(tc.path)
		if !ok {
			t.Errorf("Match(%q) expected match", tc.path)
			continue
		}
		if r.Pattern != tc.wantPat {
			t.Errorf("Match(%q) pattern=%q, want %q", tc.path, r.Pattern, tc.wantPat)
		}
		for k, wantV := range tc.wantParams {
			if !reflect.DeepEqual(params[k], wantV) {
				t.Errorf("Match(%q) params[%q]=%v, want %v", tc.path, k, params[k], wantV)
			}
		}
	}
}

// ── No match ────────────────────────────────────────────────────────────────

func TestRouteTable_NoMatch(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/posts"),
		CompilePattern("/about"),
	})

	paths := []string{"/missing", "/posts/extra", "/about/page", "/"}
	for _, p := range paths {
		_, _, ok := rt.Match(p)
		if ok {
			t.Errorf("Match(%q) expected no match", p)
		}
	}
}

// ── URL-decoded params ──────────────────────────────────────────────────────

func TestRouteTable_URLDecodedParams(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/posts/:slug"),
		CompilePattern("/files/:...path"),
	})

	r, params, ok := rt.Match("/posts/hello%20world")
	if !ok {
		t.Fatal("expected match")
	}
	if r.Pattern != "/posts/:slug" {
		t.Errorf("pattern = %q", r.Pattern)
	}
	if params["slug"] != "hello world" {
		t.Errorf("slug = %v, want 'hello world'", params["slug"])
	}

	r, params, ok = rt.Match("/files/a%20b/c%20d")
	if !ok {
		t.Fatal("expected match for encoded catch-all")
	}
	if !reflect.DeepEqual(params["path"], []string{"a b", "c d"}) {
		t.Errorf("path = %v, want [a b, c d]", params["path"])
	}
}

func TestRouteTable_URLDecodedUTF8(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/posts/:slug"),
	})

	_, params, ok := rt.Match("/posts/caf%C3%A9")
	if !ok {
		t.Fatal("expected match")
	}
	if params["slug"] != "café" {
		t.Errorf("slug = %v, want café", params["slug"])
	}
}

// ── Trailing slash normalization ────────────────────────────────────────────

func TestRouteTable_TrailingSlash(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/posts"),
		CompilePattern("/posts/:slug"),
	})

	// With trailing slash should still match.
	r, _, ok := rt.Match("/posts/")
	if !ok {
		t.Fatal("expected /posts/ to match /posts")
	}
	if r.Pattern != "/posts" {
		t.Errorf("pattern = %q, want /posts", r.Pattern)
	}

	r, params, ok := rt.Match("/posts/hello/")
	if !ok {
		t.Fatal("expected /posts/hello/ to match")
	}
	if r.Pattern != "/posts/:slug" {
		t.Errorf("pattern = %q", r.Pattern)
	}
	if params["slug"] != "hello" {
		t.Errorf("slug = %v", params["slug"])
	}
}

// ── Root with other routes ──────────────────────────────────────────────────

func TestRouteTable_RootAndOthers(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/"),
		CompilePattern("/posts"),
		CompilePattern("/posts/:slug"),
	})

	r, _, ok := rt.Match("/")
	if !ok {
		t.Fatal("expected match for /")
	}
	if r.Pattern != "/" {
		t.Errorf("pattern = %q, want /", r.Pattern)
	}

	r, _, ok = rt.Match("/posts")
	if !ok {
		t.Fatal("expected match for /posts")
	}
	if r.Pattern != "/posts" {
		t.Errorf("pattern = %q, want /posts", r.Pattern)
	}
}

// ── Complex mixed routes ────────────────────────────────────────────────────

func TestRouteTable_ComplexMixed(t *testing.T) {
	rt := NewRouteTable([]*CompiledRoute{
		CompilePattern("/"),
		CompilePattern("/api/health"),
		CompilePattern("/api/users"),
		CompilePattern("/api/users/:id"),
		CompilePattern("/api/users/:id/posts"),
		CompilePattern("/api/users/:id/posts/:postId"),
		CompilePattern("/docs/:...slug?"),
		CompilePattern("/files/:...path"),
	})

	cases := []struct {
		path       string
		wantPat    string
		wantOK     bool
		wantParams map[string]any
	}{
		{"/", "/", true, map[string]any{}},
		{"/api/health", "/api/health", true, map[string]any{}},
		{"/api/users", "/api/users", true, map[string]any{}},
		{"/api/users/42", "/api/users/:id", true, map[string]any{"id": "42"}},
		{"/api/users/42/posts", "/api/users/:id/posts", true, map[string]any{"id": "42"}},
		{"/api/users/42/posts/7", "/api/users/:id/posts/:postId", true, map[string]any{"id": "42", "postId": "7"}},
		{"/docs", "/docs/:...slug?", true, map[string]any{}},
		{"/docs/intro/setup", "/docs/:...slug?", true, map[string]any{"slug": []string{"intro", "setup"}}},
		{"/files/a/b", "/files/:...path", true, map[string]any{"path": []string{"a", "b"}}},
		{"/files", "", false, nil},
		{"/unknown", "", false, nil},
	}

	for _, tc := range cases {
		r, params, ok := rt.Match(tc.path)
		if ok != tc.wantOK {
			t.Errorf("Match(%q) ok=%v, want %v", tc.path, ok, tc.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if r.Pattern != tc.wantPat {
			t.Errorf("Match(%q) pattern=%q, want %q", tc.path, r.Pattern, tc.wantPat)
		}
		for k, wantV := range tc.wantParams {
			if !reflect.DeepEqual(params[k], wantV) {
				t.Errorf("Match(%q) params[%q]=%v, want %v", tc.path, k, params[k], wantV)
			}
		}
	}
}

// ── Empty route table ───────────────────────────────────────────────────────

func TestRouteTable_Empty(t *testing.T) {
	rt := NewRouteTable(nil)
	_, _, ok := rt.Match("/anything")
	if ok {
		t.Error("expected no match on empty table")
	}
}

// ── Benchmark: RouteTable.Match vs regex Match ──────────────────────────────

func buildBenchRoutes(n int) []*CompiledRoute {
	routes := make([]*CompiledRoute, 0, n)
	// Mix of static, dynamic, catch-all, and optional catch-all routes.
	for i := 0; i < n; i++ {
		switch i % 5 {
		case 0:
			routes = append(routes, CompilePattern(fmt.Sprintf("/static/%d/page", i)))
		case 1:
			routes = append(routes, CompilePattern(fmt.Sprintf("/dynamic/%d/:id", i)))
		case 2:
			routes = append(routes, CompilePattern(fmt.Sprintf("/nested/%d/:id/detail", i)))
		case 3:
			routes = append(routes, CompilePattern(fmt.Sprintf("/catch/%d/:...rest", i)))
		case 4:
			routes = append(routes, CompilePattern(fmt.Sprintf("/opt/%d/:...rest?", i)))
		}
	}
	return routes
}

func BenchmarkRouteTable_Match(b *testing.B) {
	routes := buildBenchRoutes(50)
	rt := NewRouteTable(routes)

	paths := []string{
		"/static/0/page",
		"/dynamic/1/hello",
		"/nested/2/42/detail",
		"/catch/3/a/b/c",
		"/opt/4",
		"/opt/4/x/y",
		"/static/45/page",
		"/dynamic/46/world",
		"/missing/route",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range paths {
			rt.Match(p)
		}
	}
}

func BenchmarkRegex_Match(b *testing.B) {
	routes := buildBenchRoutes(50)
	SortBySpecificity(routes)

	paths := []string{
		"/static/0/page",
		"/dynamic/1/hello",
		"/nested/2/42/detail",
		"/catch/3/a/b/c",
		"/opt/4",
		"/opt/4/x/y",
		"/static/45/page",
		"/dynamic/46/world",
		"/missing/route",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range paths {
			path := NormalizePath(p)
			for _, cr := range routes {
				if _, ok := Match(cr, path); ok {
					break
				}
			}
		}
	}
}
