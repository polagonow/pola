package reactsuite

import (
	"strings"
	"testing"

	"github.com/polagonow/pola/test/fixture"
)

// assertErrorBoundary checks that the Flight body contains an error row and that
// the real error message was passed through as the digest via onError.
func assertErrorBoundary(t *testing.T, body string) {
	t.Helper()
	hasErrorRow := strings.Contains(body, ":E{") || strings.Contains(body, ":E\"")
	if !hasErrorRow {
		t.Errorf("expected Flight error row (:E), got:\n%s", body[:min(len(body), 300)])
	}
	if !strings.Contains(body, "test-error") {
		t.Errorf("expected digest 'test-error' in Flight body, got:\n%s", body[:min(len(body), 300)])
	}
}

// assertCorrectErrorBoundary extends assertErrorBoundary by also verifying that
// the expected error-component module ID is referenced in the Flight body. This
// proves the *correct* innermost error boundary is mounted for the page — not
// just that some boundary fired.
func assertCorrectErrorBoundary(t *testing.T, body, moduleID string) {
	t.Helper()
	assertErrorBoundary(t, body)
	if !strings.Contains(body, moduleID) {
		t.Errorf("expected error module %q referenced in Flight body, got:\n%s", moduleID, body[:min(len(body), 400)])
	}
}

// RunErrorBoundaryTests verifies that the correct error boundary fires for each route
// and that the global error boundary is wired throughout the app.
func RunErrorBoundaryTests(t *testing.T) {
	t.Helper()

	// ── Innermost boundary per route ──────────────────────────────────────────

	t.Run("InnermostBoundaryFiresForEachRoute", func(t *testing.T) {
		cases := []struct {
			name     string
			path     string
			moduleID string
		}{
			{"root", "/?error=test-error", `"app/error"`},
			{"collection route", "/posts?error=test-error", `"app/(blog)/posts/error"`},
			{"parallel collection route", "/projects?error=test-error", `"app/(work)/projects/error"`},
			{"dynamic route", "/projects/1?error=test-error", `"app/(work)/projects/[id]/error"`},
			{"static route", "/about?error=test-error", `"app/about/error"`},
			{"authenticated static route", "/profile?error=test-error", `"app/profile/error"`},
			{"dynamic detail route", "/posts/go-react-ssr?error=test-error", `"app/(blog)/posts/[slug]/error"`},
			{"nested collection route", "/posts/go-react-ssr/revisions?error=test-error", `"app/(blog)/posts/[slug]/revisions/error"`},
			{"doubly-nested dynamic route", "/posts/go-react-ssr/revisions/v1?error=test-error", `"app/(blog)/posts/[slug]/revisions/[rev]/error"`},
			{"catch-all route", "/docs?error=test-error", `"app/docs/[[...slug]]/error"`},
		}
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, tc := range cases {
				tc := tc
				t.Run(tc.name, func(t *testing.T) {
					assertCorrectErrorBoundary(t, fixture.RSC(t, f, tc.path), tc.moduleID)
				})
			}
		})
	})

	// ── Global error boundary ─────────────────────────────────────────────────

	// The global error boundary wraps all pages as the outermost fallback.
	// Its "use client" component must be importable by the browser.
	t.Run("GlobalErrorComponentExistsInClientManifest", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			if _, ok := app.Artifacts().Output.ImportURLs["app/global-error"]; !ok {
				t.Error("global-error module missing from client manifest ImportURLs")
			}
		})
	})

	// Since GlobalError wraps all pages as the outermost __FrameworkErrorBoundary__
	// fallback, it is always serialised into the Flight stream as a module (I:) row.
	t.Run("GlobalErrorBoundaryIsReferencedInEveryPageStream", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, path := range []string{"/", "/posts", "/projects", "/about", "/profile"} {
				body := fixture.RSC(t, f, path)
				if !strings.Contains(body, `"app/global-error"`) {
					t.Errorf("page %s: global-error module not referenced in RSC stream", path)
				}
			}
		})
	})

	// ── Chunk uniqueness ──────────────────────────────────────────────────────

	// Regression: multiple error.tsx files with the same base name must not all
	// resolve to the same "error-*.js" chunk.
	t.Run("ErrorModuleChunksAreUnique", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			errorModules := []string{
				"app/error",
				"app/(blog)/posts/error",
				"app/(work)/projects/error",
				"app/(work)/projects/[id]/error",
				"app/about/error",
				"app/profile/error",
				"app/(blog)/posts/[slug]/error",
				"app/(blog)/posts/[slug]/revisions/error",
				"app/(blog)/posts/[slug]/revisions/[rev]/error",
				"app/docs/[[...slug]]/error",
			}
			seen := make(map[string]string) // chunkURL → moduleID
			for _, id := range errorModules {
				url, ok := app.Artifacts().Output.ImportURLs[id]
				if !ok {
					t.Errorf("module %q missing from ImportURLs", id)
					continue
				}
				if prev, dup := seen[url]; dup {
					t.Errorf("chunk collision: %q and %q both map to %q", prev, id, url)
				}
				seen[url] = id
			}
		})
	})
}
