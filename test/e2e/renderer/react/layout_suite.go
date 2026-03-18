package reactsuite

import (
	"testing"

	"github.com/polagonow/pola/test/fixture"
)

// RunLayoutCompositionTests verifies that layouts wrap their descendant pages
// and that Suspense resolves to real content when data is available.
func RunLayoutCompositionTests(t *testing.T) {
	t.Helper()

	// ── Root layout ───────────────────────────────────────────────────────────

	// The root layout is a server component that imports client components.
	// Their module references must appear in every page's RSC stream.
	t.Run("RootLayoutIncludesClientComponents", func(t *testing.T) {
		cases := []struct {
			name     string
			moduleID string
		}{
			{"Nav", `"components/Nav"`},
			{"ThemeToggle", `"components/ThemeToggle"`},
		}
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.RSC(t, f, "/")
			for _, tc := range cases {
				if !fixture.FlightContains(body, tc.moduleID) {
					t.Errorf("root layout missing client component module reference %s", tc.moduleID)
				}
			}
		})
	})

	// ── Route-group layout presence ───────────────────────────────────────────

	t.Run("RouteGroupLayoutWrapsDescendantPages", func(t *testing.T) {
		cases := []struct {
			path  string
			wants []string
		}{
			// (blog)/layout.tsx contributes the sidebar; (blog)/posts/layout.tsx adds the section badge.
			{"/posts", []string{"Topics", "Blog"}},
			{"/about", []string{"ℹ About"}},
			{"/profile", []string{"👤 Profile"}},
		}
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, tc := range cases {
				tc := tc
				t.Run(tc.path, func(t *testing.T) {
					body := fixture.RSC(t, f, tc.path)
					for _, want := range tc.wants {
						if !fixture.FlightContains(body, want) {
							t.Errorf("route %s: route-group layout content %q missing from RSC stream", tc.path, want)
						}
					}
				})
			}
		})
	})

	// ── Segment layout wraps child routes ─────────────────────────────────────

	t.Run("SegmentLayoutWrapsChildRoutes", func(t *testing.T) {
		cases := []struct {
			path string
			want string
		}{
			{"/posts/go-react-ssr", "← All posts"},
			{"/projects/1", "← All projects"},
		}
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, tc := range cases {
				tc := tc
				t.Run(tc.path, func(t *testing.T) {
					body := fixture.RSC(t, f, tc.path)
					if !fixture.FlightContains(body, tc.want) {
						t.Errorf("route %s: segment layout content %q missing from RSC stream", tc.path, tc.want)
					}
				})
			}
		})
	})

	// ── Suspense resolves when data is immediately available ──────────────────

	// Loading components are Suspense fallbacks that appear only when async data is
	// delayed. In tests, bridge functions resolve immediately, so real content is
	// always streamed — confirming the page is NOT stuck in the loading state.
	t.Run("SuspenseResolvesToRealContentWhenDataIsAvailable", func(t *testing.T) {
		cases := []struct {
			path string
			want string
		}{
			{"/posts", "Building SSR with Go and React"},
			{"/projects", "GoJSX"},
		}
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, tc := range cases {
				tc := tc
				t.Run(tc.path, func(t *testing.T) {
					body := fixture.RSC(t, f, tc.path)
					if !fixture.FlightContains(body, tc.want) {
						t.Errorf("route %s: Suspense not resolved — real content %q not found in RSC stream", tc.path, tc.want)
					}
				})
			}
		})
	})
}
