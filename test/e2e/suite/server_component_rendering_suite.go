package suite

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/test/fixture"
)

// RunServerComponentRenderingTests verifies RSC rendering behaviour across route types.
func RunServerComponentRenderingTests(t *testing.T) {
	t.Helper()

	t.Run("ReturnsRSCContentType", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
			req.Header.Set("Content-Type", "text/x-component")
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			ct := w.Result().Header.Get("Content-Type")
			if !strings.Contains(ct, "text/x-component") {
				t.Errorf("expected text/x-component Content-Type, got %q", ct)
			}
		})
	})

	t.Run("UnknownRouteReturns404", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
			req.Header.Set("Content-Type", "text/x-component")
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			if w.Result().StatusCode != http.StatusNotFound {
				t.Errorf("expected 404 for unregistered RSC route, got %d", w.Result().StatusCode)
			}
		})
	})

	t.Run("AllRegisteredRoutesProduceFlightOutput", func(t *testing.T) {
		paths := []string{
			"/", "/posts", "/posts/go-react-ssr", "/projects", "/projects/1",
			"/about", "/profile",
			"/posts/go-react-ssr/revisions", "/posts/go-react-ssr/revisions/v1",
			"/docs", "/docs/getting-started", "/docs/getting-started/installation",
		}
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, p := range paths {
				body := fixture.RSC(t, f, p)
				if !strings.Contains(body, "0:") {
					t.Errorf("route %s: RSC response missing Flight 0: row", p)
				}
			}
		})
	})

	// ── Index route ───────────────────────────────────────────────────────────

	t.Run("IndexRouteRendersExpectedContent", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.RSC(t, f, "/")
			// The site name "DevBlog" is split across a <span>, check each half.
			for _, want := range []string{"Dev", "Blog", "Recent posts", "Projects"} {
				if !fixture.FlightContains(body, want) {
					t.Errorf("index route Flight stream missing %q", want)
				}
			}
		})
	})

	t.Run("IndexRouteIncludesDataFetchedContent", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.RSC(t, f, "/")
			for _, want := range []string{"Building SSR with Go and React", "React Server Components Deep Dive"} {
				if !fixture.FlightContains(body, want) {
					t.Errorf("index route missing bridge-fetched content %q", want)
				}
			}
		})
	})

	// ── Collection routes ─────────────────────────────────────────────────────

	t.Run("CollectionRouteRendersItems", func(t *testing.T) {
		cases := []struct {
			path  string
			wants []string
		}{
			{"/posts", []string{"Building SSR with Go and React", "Goja VM Internals", "go-react-ssr"}},
			{"/projects", []string{"GoJSX", "GojaBridge", "FlightDecode"}},
		}
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, tc := range cases {
				tc := tc
				t.Run(tc.path, func(t *testing.T) {
					body := fixture.RSC(t, f, tc.path)
					for _, want := range tc.wants {
						if !fixture.FlightContains(body, want) {
							t.Errorf("collection route %s missing item %q", tc.path, want)
						}
					}
				})
			}
		})
	})

	t.Run("CollectionRouteSupportsQueryParamFiltering", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/posts?tag=go", nil)
			req.Header.Set("Content-Type", "text/x-component")
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			if w.Result().StatusCode != http.StatusOK {
				t.Fatalf("RSC GET /posts?tag=go → %d", w.Result().StatusCode)
			}
			body, _ := io.ReadAll(w.Result().Body)
			if !fixture.FlightContains(string(body), "go") {
				t.Error("filtered collection route result missing expected tag content")
			}
		})
	})

	// ── Dynamic routes ────────────────────────────────────────────────────────

	t.Run("DynamicRouteRendersMatchedItem", func(t *testing.T) {
		cases := []struct {
			path  string
			wants []string
		}{
			{"/posts/go-react-ssr", []string{"Building SSR with Go and React", "Jane Doe"}},
			{"/projects/1", []string{"GoJSX", "142", "active"}},
		}
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, tc := range cases {
				tc := tc
				t.Run(tc.path, func(t *testing.T) {
					body := fixture.RSC(t, f, tc.path)
					for _, want := range tc.wants {
						if !fixture.FlightContains(body, want) {
							t.Errorf("dynamic route %s missing %q", tc.path, want)
						}
					}
				})
			}
		})
	})

	// ── Nested routes ─────────────────────────────────────────────────────────

	t.Run("NestedCollectionRouteRendersItems", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.RSC(t, f, "/posts/go-react-ssr/revisions")
			for _, want := range []string{"v3", "v2", "v1", "Published", "Initial draft"} {
				if !fixture.FlightContains(body, want) {
					t.Errorf("nested collection route missing %q", want)
				}
			}
		})
	})

	t.Run("DoublyNestedDynamicRouteRendersItem", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.RSC(t, f, "/posts/go-react-ssr/revisions/v2")
			for _, want := range []string{"v2", "Building SSR with Go and React", "esbuild two-pass"} {
				if !fixture.FlightContains(body, want) {
					t.Errorf("doubly-nested dynamic route missing %q", want)
				}
			}
		})
	})

	// ── Static routes ─────────────────────────────────────────────────────────

	t.Run("StaticRouteRendersContent", func(t *testing.T) {
		cases := []struct {
			path  string
			wants []string
		}{
			{"/about", []string{"About", "GoJSX", "Goja"}},
			{"/profile", []string{"Jane Doe", "jane@example.com", "Senior Engineer"}},
		}
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, tc := range cases {
				tc := tc
				t.Run(tc.path, func(t *testing.T) {
					body := fixture.RSC(t, f, tc.path)
					for _, want := range tc.wants {
						if !fixture.FlightContains(body, want) {
							t.Errorf("static route %s missing %q", tc.path, want)
						}
					}
				})
			}
		})
	})

	t.Run("RouteReceivesSearchParams", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/profile?id=1", nil)
			req.Header.Set("Content-Type", "text/x-component")
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			body, _ := io.ReadAll(w.Result().Body)
			if !fixture.FlightContains(string(body), "Jane Doe") {
				t.Error("route did not receive search params: expected bridge-fetched content missing")
			}
		})
	})

	// ── Optional catch-all route ──────────────────────────────────────────────

	t.Run("OptionalCatchAllRouteHandlesZeroSegments", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.RSC(t, f, "/docs")
			for _, want := range []string{"Documentation", "Getting Started", "Core Concepts"} {
				if !fixture.FlightContains(body, want) {
					t.Errorf("optional catch-all index missing %q", want)
				}
			}
		})
	})

	t.Run("OptionalCatchAllRouteHandlesOneSegment", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.RSC(t, f, "/docs/getting-started")
			if !fixture.FlightContains(body, "getting-started") && !fixture.FlightContains(body, "Getting Started") {
				t.Error("optional catch-all route with one segment missing expected section content")
			}
		})
	})

	t.Run("OptionalCatchAllRouteHandlesMultipleSegments", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.RSC(t, f, "/docs/getting-started/installation")
			if !fixture.FlightContains(body, "installation") {
				t.Error("optional catch-all route with multiple segments missing expected leaf content")
			}
		})
	})
}
