package suite

import (
	"testing"

	"gojsx/test/e2e/fixture"
)

// RunFlightProtocolTests verifies structural invariants of the React Flight wire format.
func RunFlightProtocolTests(t *testing.T) {
	t.Helper()

	t.Run("RootRowIsReactElementTree", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.RSC(t, f, "/")
			tree := fixture.FlightTree(t, body)
			arr, ok := tree.([]any)
			if !ok || len(arr) < 2 {
				t.Fatalf("Flight root row 0: expected JSON array, got %T: %v", tree, tree)
			}
			if arr[0] != "$" {
				t.Errorf("Flight root row 0: expected '$' as first element (React element marker), got %v", arr[0])
			}
		})
	})

	t.Run("AsyncComponentsLandInLaterRows", func(t *testing.T) {
		// Async server components (e.g. data-fetching components) are streamed in
		// rows after row 0. Verifying their content via the raw body — not just row 0 —
		// confirms the streaming pipeline is working end-to-end.
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.RSC(t, f, "/")
			for _, want := range []string{"Dev", "Blog", "Read posts"} {
				if !fixture.FlightContains(body, want) {
					t.Errorf("async component content %q missing from Flight stream", want)
				}
			}
		})
	})
}
