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

// assertNotFoundError verifies that a Flight body contains an error row and
// that the error digest contains "not found" — confirming the page threw a
// data-missing error rather than an unrelated server error.
func assertNotFoundError(t *testing.T, body string) {
	t.Helper()
	if !strings.Contains(body, ":E{") && !strings.Contains(body, ":E\"") {
		t.Errorf("expected Flight error row (:E), got:\n%s", body[:min(len(body), 300)])
	}
	if !strings.Contains(strings.ToLower(body), "not found") {
		t.Errorf("expected \"not found\" in error digest, got:\n%s", body[:min(len(body), 300)])
	}
}

// RunNotFoundHandlingTests verifies 404 and data-missing error behaviour.
func RunNotFoundHandlingTests(t *testing.T) {
	t.Helper()

	// ── Unmatched route (no route registered for the path) ────────────────────

	t.Run("UnmatchedRouteRSCReturns404", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			status, _ := fixture.RSCAny(t, f, "/nonexistent-route")
			if status != http.StatusNotFound {
				t.Errorf("expected HTTP 404 for unmatched RSC route, got %d", status)
			}
		})
	})

	t.Run("UnmatchedRouteRSCReturnsRSCContentType", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/nonexistent-route", nil)
			req.Header.Set("Content-Type", "text/x-component")
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			ct := w.Result().Header.Get("Content-Type")
			if !strings.Contains(ct, "text/x-component") {
				t.Errorf("expected text/x-component for RSC 404 response, got %q", ct)
			}
		})
	})

	t.Run("UnmatchedRouteRSCResponseContainsFlightRow", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			_, body := fixture.RSCAny(t, f, "/nonexistent-route")
			if !strings.Contains(body, "0:") {
				t.Errorf("unmatched route RSC response missing Flight 0: row, got:\n%s", body[:min(len(body), 300)])
			}
		})
	})

	t.Run("UnmatchedRouteRSCResponseRendersNotFoundComponent", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			_, body := fixture.RSCAny(t, f, "/nonexistent-route")
			for _, want := range []string{"Page not found", "Go home"} {
				if !strings.Contains(body, want) {
					t.Errorf("unmatched route RSC response missing %q, got:\n%s", want, body[:min(len(body), 400)])
				}
			}
		})
	})

	t.Run("UnmatchedRouteHTMLReturns404", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/nonexistent-route", nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			if w.Result().StatusCode != http.StatusNotFound {
				t.Errorf("expected HTTP 404 for unmatched HTML route, got %d", w.Result().StatusCode)
			}
		})
	})

	t.Run("UnmatchedRouteHTMLResponseIsClientShell", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/nonexistent-route", nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			body, _ := io.ReadAll(w.Result().Body)
			// The HTML 404 response is the standard client shell (not a raw error page).
			// The browser bootstraps and makes a subsequent RSC request to fetch the
			// actual GlobalNotFound component content.
			if !strings.Contains(string(body), `id="__POLA_ROOT__"`) {
				t.Errorf("unmatched route HTML response should be client shell with id=\"__POLA_ROOT__\", got:\n%s",
					string(body)[:min(len(string(body)), 400)])
			}
		})
	})

	// ── Matched route with missing data (error boundary fires) ────────────────

	t.Run("DataMissingTriggersErrorBoundaryWithNotFoundDigest", func(t *testing.T) {
		cases := []struct {
			name string
			path string
		}{
			{"dynamic param not in data store", "/posts/no-such-post"},
			{"nested dynamic param not in data store", "/projects/no-such-id"},
			{"doubly-nested dynamic param not in data store", "/posts/go-react-ssr/revisions/v99"},
		}
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			for _, tc := range cases {
				tc := tc
				t.Run(tc.name, func(t *testing.T) {
					assertNotFoundError(t, fixture.RSC(t, f, tc.path))
				})
			}
		})
	})
}
