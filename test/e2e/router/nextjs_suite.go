// Package routersuite contains router-specific e2e test suites.
package routersuite

import (
	"net/http"
	"testing"

	"github.com/polagonow/pola/test/fixture"
)

// RunNextJSRouterTests verifies Next.js-style file-system routing: which URL
// maps to which page, how dynamic/catch-all segments are captured, that route
// groups are transparent in the URL, and that unmatched paths 404. It runs
// against every React app fixture (RSC rendering is required to observe the
// captured params in the Flight output).
//
// These assertions focus on route *matching* semantics; the rendered content
// of each route is covered by suite.RunServerComponentRenderingTests.
func RunNextJSRouterTests(t *testing.T) {
	t.Helper()

	// ── Matching ──────────────────────────────────────────────────────────────

	t.Run("IndexRouteMatches", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			assertStatus(t, f, "/", http.StatusOK)
		})
	})

	// The (blog) and (work) directories are Next.js route groups: they organise
	// files without appearing in the URL. /posts and /projects must resolve even
	// though they live under app/(blog)/ and app/(work)/.
	t.Run("RouteGroupsAreTransparentInURL", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			assertStatus(t, f, "/posts", http.StatusOK)
			assertStatus(t, f, "/projects", http.StatusOK)
			// The literal group segment must NOT be routable.
			assertStatus(t, f, "/(blog)/posts", http.StatusNotFound)
		})
	})

	t.Run("StaticRoutesMatchExactly", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			assertStatus(t, f, "/about", http.StatusOK)
			assertStatus(t, f, "/profile", http.StatusOK)
			// A near-miss on a static segment must not match.
			assertStatus(t, f, "/abou", http.StatusNotFound)
			assertStatus(t, f, "/aboutx", http.StatusNotFound)
		})
	})

	// ── Dynamic segments ────────────────────────────────────────────────────────

	// The [slug] segment is captured and routed: two different slugs resolve to
	// their respective posts.
	t.Run("DynamicSegmentSelectsMatchingItem", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			a := fixture.RSC(t, f, "/posts/go-react-ssr")
			if !fixture.FlightContains(a, "Building SSR with Go and React") {
				t.Error("/posts/go-react-ssr did not render the matching post")
			}
			b := fixture.RSC(t, f, "/posts/rsc-deep-dive")
			if !fixture.FlightContains(b, "React Server Components Deep Dive") {
				t.Error("/posts/rsc-deep-dive did not render the matching post")
			}
			// The two dynamic matches must produce distinct content.
			if fixture.FlightContains(a, "React Server Components Deep Dive") {
				t.Error("dynamic route did not distinguish slugs")
			}
		})
	})

	t.Run("DynamicParamDistinguishesSiblingRoutes", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			one := fixture.RSC(t, f, "/projects/1")
			two := fixture.RSC(t, f, "/projects/2")
			if !fixture.FlightContains(one, "GoJSX") {
				t.Error("/projects/1 missing GoJSX")
			}
			if !fixture.FlightContains(two, "GojaBridge") {
				t.Error("/projects/2 missing GojaBridge")
			}
		})
	})

	// ── Nested routes ────────────────────────────────────────────────────────────

	t.Run("NestedRoutesMatchAndCaptureParams", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			// Nested collection: /posts/[slug]/revisions.
			assertStatus(t, f, "/posts/go-react-ssr/revisions", http.StatusOK)
			// Doubly-nested dynamic: /posts/[slug]/revisions/[rev]. Both the
			// outer slug and inner rev must be captured to resolve the leaf.
			leaf := fixture.RSC(t, f, "/posts/go-react-ssr/revisions/v2")
			if !fixture.FlightContains(leaf, "v2") {
				t.Error("nested dynamic route did not capture the [rev] param")
			}
		})
	})

	// ── Optional catch-all ─────────────────────────────────────────────────────

	// app/docs/[[...slug]] must match zero, one, and many trailing segments.
	t.Run("OptionalCatchAllMatchesVariableDepth", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, path := range []string{
				"/docs",
				"/docs/getting-started",
				"/docs/getting-started/installation",
			} {
				assertStatus(t, f, path, http.StatusOK)
			}
		})
	})

	// ── Non-matches ────────────────────────────────────────────────────────────

	// Paths with no matching file (including extra segments beyond a static or
	// single-dynamic leaf) must 404. Note these are structural non-matches, not
	// valid routes whose data lookup fails.
	t.Run("UnmatchedRoutesReturn404", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, path := range []string{
				"/nonexistent",
				"/about/extra",      // extra segment past a static leaf
				"/profile/1/2",      // ditto
				"/projects/1/extra", // extra segment past a single-dynamic leaf
			} {
				assertStatus(t, f, path, http.StatusNotFound)
			}
		})
	})
}

// assertStatus makes an RSC request and asserts the HTTP status code.
func assertStatus(t *testing.T, f fixture.AppFixture, path string, want int) {
	t.Helper()
	got, _ := fixture.RSCAny(t, f, path)
	if got != want {
		t.Errorf("RSC GET %s → %d, want %d", path, got, want)
	}
}
