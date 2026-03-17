package e2etest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRSC_ContentType(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		app := f.getApp(t)
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
		req.Header.Set("Content-Type", "text/x-component")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		ct := w.Result().Header.Get("Content-Type")
		if !strings.Contains(ct, "text/x-component") {
			t.Errorf("expected text/x-component, got %q", ct)
		}
	})
}

func TestRSC_404(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		app := f.getApp(t)
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
		req.Header.Set("Content-Type", "text/x-component")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 for unknown route, got %d", w.Result().StatusCode)
		}
	})
}

func TestRSC_AllRoutesRender(t *testing.T) {
	paths := []string{"/", "/posts", "/posts/go-react-ssr", "/projects", "/projects/1", "/about", "/profile",
		"/posts/go-react-ssr/revisions", "/posts/go-react-ssr/revisions/v1",
		"/docs", "/docs/getting-started", "/docs/getting-started/installation"}
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		for _, p := range paths {
			body := RSC(t, f, p)
			if !strings.Contains(body, "0:") {
				t.Errorf("page %s: missing Flight 0: row", p)
			}
		}
	})
}

// ─── Home ──────────────────────────────────────────────────────────────────────

func TestRSC_HomePage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/")
		// "DevBlog" is split across a <span> in the JSX, so check the two halves separately.
		for _, want := range []string{"Dev", "Blog", "Recent posts", "Projects"} {
			if !FlightContains(body, want) {
				t.Errorf("HomePage Flight missing %q", want)
			}
		}
	})
}

func TestRSC_HomePage_HasPosts(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/")
		for _, want := range []string{"Building SSR with Go and React", "React Server Components Deep Dive"} {
			if !FlightContains(body, want) {
				t.Errorf("HomePage missing post %q", want)
			}
		}
	})
}

// ─── Posts ─────────────────────────────────────────────────────────────────────

func TestRSC_PostsPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/posts")
		for _, want := range []string{"Building SSR with Go and React", "Goja VM Internals", "go-react-ssr"} {
			if !FlightContains(body, want) {
				t.Errorf("PostsPage missing %q", want)
			}
		}
	})
}

func TestRSC_PostsPage_TagFilter(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		app := f.getApp(t)
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/posts?tag=go", nil)
		req.Header.Set("Content-Type", "text/x-component")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("RSC GET /posts?tag=go → %d", w.Result().StatusCode)
		}
		body, _ := io.ReadAll(w.Result().Body)
		if !FlightContains(string(body), "go") {
			t.Error("tag filter result missing 'go'")
		}
	})
}

func TestRSC_PostDetailPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/posts/go-react-ssr")
		for _, want := range []string{"Building SSR with Go and React", "Jane Doe"} {
			if !FlightContains(body, want) {
				t.Errorf("PostDetail missing %q", want)
			}
		}
	})
}

func TestRSC_PostDetailPage_NotFound(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		app := f.getApp(t)
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/posts/nonexistent-slug", nil)
		req.Header.Set("Content-Type", "text/x-component")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		body, _ := io.ReadAll(w.Result().Body)
		if !strings.Contains(string(body), ":E{") && !strings.Contains(string(body), ":E\"") {
			t.Errorf("expected Flight error row for missing post, got: %s", string(body)[:min(len(string(body)), 200)])
		}
	})
}

// ─── Revisions ─────────────────────────────────────────────────────────────────

func TestRSC_RevisionsPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/posts/go-react-ssr/revisions")
		for _, want := range []string{"v3", "v2", "v1", "Published", "Initial draft"} {
			if !FlightContains(body, want) {
				t.Errorf("RevisionsPage missing %q", want)
			}
		}
	})
}

func TestRSC_RevisionDetailPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/posts/go-react-ssr/revisions/v2")
		for _, want := range []string{"v2", "Building SSR with Go and React", "esbuild two-pass"} {
			if !FlightContains(body, want) {
				t.Errorf("RevisionDetail missing %q", want)
			}
		}
	})
}

func TestRSC_RevisionDetailPage_NotFound(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		app := f.getApp(t)
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/posts/go-react-ssr/revisions/v99", nil)
		req.Header.Set("Content-Type", "text/x-component")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		body, _ := io.ReadAll(w.Result().Body)
		if !strings.Contains(string(body), ":E{") && !strings.Contains(string(body), ":E\"") {
			t.Errorf("expected Flight error row for missing revision, got: %s", string(body)[:min(len(string(body)), 200)])
		}
	})
}

// ─── Projects ──────────────────────────────────────────────────────────────────

func TestRSC_ProjectsPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/projects")
		for _, want := range []string{"GoJSX", "GojaBridge", "FlightDecode"} {
			if !FlightContains(body, want) {
				t.Errorf("ProjectsPage missing %q", want)
			}
		}
	})
}

func TestRSC_ProjectDetailPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/projects/1")
		for _, want := range []string{"GoJSX", "142", "active"} {
			if !FlightContains(body, want) {
				t.Errorf("ProjectDetail missing %q", want)
			}
		}
	})
}

func TestRSC_ProjectDetailPage_NotFound(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		app := f.getApp(t)
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/projects/999", nil)
		req.Header.Set("Content-Type", "text/x-component")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		body, _ := io.ReadAll(w.Result().Body)
		if !strings.Contains(string(body), ":E{") && !strings.Contains(string(body), ":E\"") {
			t.Errorf("expected Flight error row for missing project, got: %s", string(body)[:min(len(string(body)), 200)])
		}
	})
}

// ─── About ─────────────────────────────────────────────────────────────────────

func TestRSC_AboutPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/about")
		for _, want := range []string{"About", "GoJSX", "Goja"} {
			if !FlightContains(body, want) {
				t.Errorf("AboutPage missing %q", want)
			}
		}
	})
}

// ─── Profile ───────────────────────────────────────────────────────────────────

func TestRSC_ProfilePage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/profile")
		for _, want := range []string{"Jane Doe", "jane@example.com", "Senior Engineer"} {
			if !FlightContains(body, want) {
				t.Errorf("ProfilePage missing %q", want)
			}
		}
	})
}

func TestRSC_ProfilePage_SearchParams(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		app := f.getApp(t)
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/profile?id=1", nil)
		req.Header.Set("Content-Type", "text/x-component")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		body, _ := io.ReadAll(w.Result().Body)
		if !FlightContains(string(body), "Jane Doe") {
			t.Error("ProfilePage with searchParams missing 'Jane Doe'")
		}
	})
}

// ─── Docs (catch-all route) ────────────────────────────────────────────────────

func TestRSC_DocsIndex(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/docs")
		for _, want := range []string{"Documentation", "Getting Started", "Core Concepts"} {
			if !FlightContains(body, want) {
				t.Errorf("DocsIndex missing %q", want)
			}
		}
	})
}

func TestRSC_DocsSection(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/docs/getting-started")
		if !FlightContains(body, "getting-started") && !FlightContains(body, "Getting Started") {
			t.Error("DocsSection missing section name")
		}
	})
}

func TestRSC_DocsNestedPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/docs/getting-started/installation")
		if !FlightContains(body, "installation") {
			t.Error("DocsNestedPage missing 'installation'")
		}
	})
}
